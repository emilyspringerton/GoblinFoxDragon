package main

// SSH_TRANSPORT_IDENTITY_SPEC.md §2 / Stage 4 (docs2/SSH_TRANSPORT_IDENTITY_NORTHSTAR.md):
// "Game SSH server" -- a real SSH listener on a high port, public-key-only, trust-on-first-use
// (any offered key is accepted for the CONNECTION; binding a key to a specific character/account
// is §3, Stage 5, not built here). No identity yet means every SSH session today is exactly as
// anonymous as a telnet one -- see handleConn's own player struct literal, reused completely
// unchanged for both transports, which is why `isGuest: true` there already applies correctly to
// SSH sessions with zero extra code: an SSH connection with no bound identity is still a guest,
// per the spec's own explicit framing (see the player struct's `isGuest` field doc comment).
//
// Design choice, checked against this file before writing a line of this: `handle()`'s own
// command dispatch and `handleConn`'s own read/write loop only ever call four methods on
// `p.conn` (net.Conn) anywhere in this file -- Read, Write, Close, and RemoteAddr (grepped:
// zero SetDeadline/SetReadDeadline/SetWriteDeadline/LocalAddr calls exist). That means an
// `ssh.Channel` (Read/Write/Close, plus SendRequest/CloseWrite/Stderr that nothing here ever
// calls) needs only RemoteAddr/LocalAddr/three deadline no-ops bolted on to satisfy net.Conn --
// sshConnAdapter below -- and `handleConn(adapter)` runs completely unmodified, same struct
// literal, same guest gate, same IDUNA fetch-or-create, same disconnect sync. This is the real,
// positive finding the NORTHSTAR doc named after Stage 1/2/3: the dispatch layer really is
// transport-agnostic, exercised for real here for the first time.

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

const (
	// sshHostKeyPath (§2.2: "Generated once, persisted outside the repo, included in deploy
	// secrets and backups. Regenerating it triggers MITM warnings for every returning client.
	// Treat as permanent.") -- var/ is this process's one real ReadWritePaths data path
	// (ops/systemd/gfd-mud.service, Stage 3) and already git-ignored, so a host key generated
	// here is never committed and is already covered by the new `gfd` emily-backup target
	// (Stage 3) without any extra wiring.
	sshHostKeyPath = "var/ssh_host_ed25519_key"

	// sshIdleTimeout / sshMaxSessionDuration (§2.4 resource limits): telnet's own handleConn has
	// no equivalent today (grepped: zero deadline calls anywhere in this file, sessions only end
	// on a real TCP close) -- SSH gets real enforcement here since the spec asks for it
	// specifically in §2.4, not because telnet already has it. Values are generous on purpose:
	// this is abuse containment (a forgotten connection, a hung client), not a gameplay-session
	// cap -- 6 hours covers a real, long play session without special-casing anything.
	sshIdleTimeout      = 15 * time.Minute
	sshMaxSessionDur    = 6 * time.Hour
	sshWatchdogInterval = 30 * time.Second
	sshMaxSessionsPerIP = 5 // §2.4 "max concurrent sessions per source IP"
	sshMaxAuthTries     = 3 // §2.4 "auth attempt limit per connection" (TOFU means the first
	// publickey attempt always succeeds; this only ever bites a client stuck retrying an
	// unsupported method like password/keyboard-interactive, both refused below)
)

// sshPerIPSessions tracks live SSH session counts per source IP for the §2.4 concurrent-session
// cap. Guarded by its own mutex rather than gw.mu -- this has nothing to do with game state, and
// taking the game-world lock for connection accounting would be a real, unnecessary contention
// point on every single connect/disconnect.
var (
	sshPerIPMu       sync.Mutex
	sshPerIPSessions = map[string]int{}
)

// loadOrGenerateSSHHostKey implements §2.2 literally: load the persisted key if one exists,
// otherwise generate a real Ed25519 keypair (modern, fast, short -- the same real reasoning
// OpenSSH itself defaults new host keys to today) and persist it before returning, so every
// restart after the first reuses the exact same key. 0600 permissions match the existing
// ~/.config/gfd-mud/env convention (idunaclient secrets) for "a real private key file on disk."
func loadOrGenerateSSHHostKey(path string) (ssh.Signer, error) {
	if raw, err := os.ReadFile(path); err == nil {
		signer, err := ssh.ParsePrivateKey(raw)
		if err != nil {
			return nil, fmt.Errorf("ssh: parse existing host key at %s: %w", path, err)
		}
		return signer, nil
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("ssh: read host key at %s: %w", path, err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ssh: generate host key: %w", err)
	}
	_ = pub // only the private key is persisted; ssh.NewSignerFromKey derives the public half.

	block, err := ssh.MarshalPrivateKey(priv, "gfd-mud ssh host key, generated "+time.Now().UTC().Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("ssh: marshal new host key: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("ssh: mkdir for host key: %w", err)
	}
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0600); err != nil {
		return nil, fmt.Errorf("ssh: persist new host key: %w", err)
	}
	log.Printf("[ssh] generated new host key at %s (first boot with SSH enabled)", path)

	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		return nil, fmt.Errorf("ssh: derive signer from new host key: %w", err)
	}
	return signer, nil
}

// sshConnAdapter makes an ssh.Channel satisfy net.Conn -- see this file's own top-of-file comment
// for why this is enough for handleConn to run completely unmodified. Also tracks last-activity
// (bumped on every real Read) for the idle-timeout watchdog below.
type sshConnAdapter struct {
	ssh.Channel
	underlying net.Conn // the real TCP conn, for RemoteAddr/LocalAddr only -- never read/written directly
	lastActive int64    // unix nanoseconds, atomic
}

func newSSHConnAdapter(ch ssh.Channel, underlying net.Conn) *sshConnAdapter {
	return &sshConnAdapter{Channel: ch, underlying: underlying, lastActive: time.Now().UnixNano()}
}

func (a *sshConnAdapter) Read(p []byte) (int, error) {
	n, err := a.Channel.Read(p)
	if n > 0 {
		atomic.StoreInt64(&a.lastActive, time.Now().UnixNano())
	}
	return n, err
}

func (a *sshConnAdapter) LocalAddr() net.Addr  { return a.underlying.LocalAddr() }
func (a *sshConnAdapter) RemoteAddr() net.Addr { return a.underlying.RemoteAddr() }

// SetDeadline/SetReadDeadline/SetWriteDeadline are no-ops: telnet's own handleConn never calls
// any of them either (see this file's top-of-file comment), so this preserves identical
// behavior across both transports. Idle/max-session enforcement is a separate watchdog
// (sshSessionWatchdog) that closes the channel directly rather than relying on socket deadlines.
func (a *sshConnAdapter) SetDeadline(t time.Time) error      { return nil }
func (a *sshConnAdapter) SetReadDeadline(t time.Time) error  { return nil }
func (a *sshConnAdapter) SetWriteDeadline(t time.Time) error { return nil }

// sshSessionWatchdog enforces §2.4's idle-timeout and max-session-duration limits by force-
// closing the channel -- handleConn's own blocked Read then returns a real error and its existing
// disconnect/IDUNA-sync path runs exactly as it would for any other dropped connection. Returns
// (rather than leaking) as soon as `done` closes, which the caller does right after handleConn
// itself returns for any other reason.
func sshSessionWatchdog(adapter *sshConnAdapter, done <-chan struct{}) {
	start := time.Now()
	ticker := time.NewTicker(sshWatchdogInterval)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			idleFor := time.Since(time.Unix(0, atomic.LoadInt64(&adapter.lastActive)))
			if idleFor > sshIdleTimeout {
				log.Printf("[ssh] idle timeout (%s) closing session remote=%s", idleFor.Round(time.Second), adapter.RemoteAddr())
				_ = adapter.Close()
				return
			}
			if time.Since(start) > sshMaxSessionDur {
				log.Printf("[ssh] max session duration reached closing session remote=%s", adapter.RemoteAddr())
				_ = adapter.Close()
				return
			}
		}
	}
}

// handleSSHChannels services every "session" channel opened on one already-authenticated SSH
// connection (a client may open more than one, e.g. a second shell -- each gets its own
// handleConn/player, same as two separate telnet connections would). Non-"session" channel types
// (direct-tcpip, etc.) are rejected outright -- this is a game server, not a general SSH gateway.
func handleSSHChannels(chans <-chan ssh.NewChannel, underlying net.Conn) {
	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			_ = newChannel.Reject(ssh.UnknownChannelType, "only session channels are supported")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("[ssh] channel accept failed remote=%s: %v", underlying.RemoteAddr(), err)
			continue
		}

		go func() {
			adapter := newSSHConnAdapter(channel, underlying)
			started := false
			for req := range requests {
				switch req.Type {
				case "pty-req", "window-change":
					// §2.3: honor PTY requests (initial size + resize) by acknowledging them --
					// this MUD's own output (both telnet and now SSH) is a plain, unwrapped
					// text stream with no width-aware reflow anywhere in this file (checked: no
					// terminal-width state exists today), so there is no real geometry to apply
					// yet. Acked rather than rejected so well-behaved clients (which wait for a
					// reply before proceeding) never stall -- real, honest scope: "does not
					// crash or hang on this request," not "reflows text to the new width."
					if req.WantReply {
						_ = req.Reply(true, nil)
					}
				case "shell":
					// The one request type that actually starts the game session -- matches
					// real sshd behavior (pty-req then shell for an interactive login), and
					// degrades cleanly for a client that sends "shell" with no preceding
					// pty-req at all (§2.3 "degrade cleanly with no PTY").
					if req.WantReply {
						_ = req.Reply(true, nil)
					}
					if !started {
						started = true
						done := make(chan struct{})
						go sshSessionWatchdog(adapter, done)
						handleConn(adapter)
						close(done)
					}
				default:
					// "exec", subsystem requests, etc. -- a real, clean rejection rather than
					// silence (§2.3's own "no crash, no hang" bar applies to malformed/
					// unsupported requests too, not only to the PTY-vs-no-PTY case).
					if req.WantReply {
						_ = req.Reply(false, nil)
					}
				}
			}
		}()
	}
}

// startSSHListener implements §2.1's transport requirements end to end: public-key-only
// (PasswordCallback/KeyboardInteractiveCallback deliberately left nil, so both auth methods are
// simply unsupported -- refused, not merely discouraged), trust-on-first-use (PublicKeyCallback
// accepts any syntactically valid offered key unconditionally; binding a key to a character is
// §3/Stage 5, not built here), and ignores whatever username the client sends (never inspected,
// never causes an auth failure).
func startSSHListener(port int) {
	hostKey, err := loadOrGenerateSSHHostKey(sshHostKeyPath)
	if err != nil {
		log.Printf("[ssh] disabled: %v", err)
		return
	}
	config := buildSSHServerConfig(hostKey)

	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("[ssh] listen failed on %s: %v", addr, err)
		return
	}
	log.Printf("[ssh] listening on %s (public-key only, TOFU, no identity binding yet -- Stage 4 of SSH_TRANSPORT_IDENTITY_SPEC.md)", addr)
	serveSSH(ln, config)
}

// buildSSHServerConfig is split out from startSSHListener purely so ssh_listener_test.go can
// exercise the real §2.1 auth semantics (password refused, public-key/TOFU accepted) directly
// against a real ssh.NewServerConn handshake over a real loopback TCP connection -- no game-world
// (`gw`) initialization required, since no "session" channel is ever opened in that test path.
func buildSSHServerConfig(hostKey ssh.Signer) *ssh.ServerConfig {
	config := &ssh.ServerConfig{
		MaxAuthTries: sshMaxAuthTries,
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			// TOFU (§2.1): accept unconditionally. conn.User() is deliberately never read here --
			// §2.1 requires the server "accept and ignore the SSH username, or treat it as a
			// character-name hint... must not error," and there is no character-name-hint
			// feature built yet (that belongs with §3/Stage 5's real identity binding), so
			// "ignore" is the accurate, honest behavior today, not "hint, half-wired."
			return &ssh.Permissions{}, nil
		},
		ServerVersion: "SSH-2.0-DragonsNShit",
	}
	config.AddHostKey(hostKey)
	return config
}

// serveSSH runs the real accept loop against an already-bound listener -- split out from
// startSSHListener so tests can pass a `net.Listen("tcp", "127.0.0.1:0")` listener (OS-assigned
// free port) instead of a hardcoded one.
func serveSSH(ln net.Listener, config *ssh.ServerConfig) {
	for {
		rawConn, err := ln.Accept()
		if err != nil {
			log.Printf("[ssh] accept: %v", err)
			continue
		}

		ip := ipOnly(rawConn.RemoteAddr())
		sshPerIPMu.Lock()
		count := sshPerIPSessions[ip]
		if count >= sshMaxSessionsPerIP {
			sshPerIPMu.Unlock()
			log.Printf("[ssh] refusing connection from %s: %d/%d concurrent sessions already open", ip, count, sshMaxSessionsPerIP)
			_ = rawConn.Close()
			continue
		}
		sshPerIPSessions[ip] = count + 1
		sshPerIPMu.Unlock()

		go func() {
			defer func() {
				sshPerIPMu.Lock()
				sshPerIPSessions[ip]--
				if sshPerIPSessions[ip] <= 0 {
					delete(sshPerIPSessions, ip)
				}
				sshPerIPMu.Unlock()
			}()

			sshConn, chans, reqs, err := ssh.NewServerConn(rawConn, config)
			if err != nil {
				// Real, expected traffic here, not just attacks: a plain TCP health check, a
				// port scanner, or a client that only speaks an unsupported auth method will
				// all end up here having never reached PublicKeyCallback successfully -- logged
				// at a real level (not silently swallowed, per §5's own "structured logs: auth
				// outcomes" ask) but this is routine noise, not necessarily hostile.
				log.Printf("[ssh] handshake failed remote=%s: %v", rawConn.RemoteAddr(), err)
				return
			}
			defer sshConn.Close()
			log.Printf("[ssh] connected remote=%s client_version=%q", rawConn.RemoteAddr(), sshConn.ClientVersion())

			go ssh.DiscardRequests(reqs)
			handleSSHChannels(chans, rawConn)
		}()
	}
}

// ipOnly strips the port from a net.Addr's string form for per-source-IP session accounting --
// falls back to the full string if it doesn't parse as host:port (defensive, not expected to
// trigger against a real net.Conn.RemoteAddr()).
func ipOnly(addr net.Addr) string {
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}
