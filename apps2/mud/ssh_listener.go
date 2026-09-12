package main

// SSH_TRANSPORT_IDENTITY_SPEC.md §2/§3 / Stages 4-5 (docs2/SSH_TRANSPORT_IDENTITY_NORTHSTAR.md):
// "Game SSH server" (§2, Stage 4) -- a real SSH listener on a high port, public-key-only,
// trust-on-first-use for the raw TCP/SSH handshake itself (any syntactically valid key is
// accepted for the CONNECTION). "Identity" (§3, Stage 5) -- resolveSSHIdentity/runSSHClaimFlow
// below turn that connection into either a resumed, permanently-bound character (isGuest=false)
// or a fresh one claimed on the spot, falling back to an anonymous guest (isGuest=true, same
// treatment telnet gets) only if IDUNA is unreachable or the player abandons the claim prompt.
//
// Design choice, checked against this file before writing a line of Stage 4: `handle()`'s own
// command dispatch and `handleConn`'s own read/write loop only ever call four methods on
// `p.conn` (net.Conn) anywhere in this file -- Read, Write, Close, and RemoteAddr (grepped:
// zero SetDeadline/SetReadDeadline/SetWriteDeadline/LocalAddr calls exist). That means an
// `ssh.Channel` (Read/Write/Close, plus SendRequest/CloseWrite/Stderr that nothing here ever
// calls) needs only RemoteAddr/LocalAddr/three deadline no-ops bolted on to satisfy net.Conn --
// sshConnAdapter below -- and `handleConn(adapter, isGuest, preset, echo)` runs the exact same struct
// literal, IDUNA fetch-or-create, and disconnect sync a telnet connection does, for both guest
// and identified SSH sessions alike. This is the real, positive finding the NORTHSTAR doc named
// after Stage 1/2/3: the dispatch layer really is transport-agnostic.

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"

	"dragonsnshit/server/idunaclient"
	"dragonsnshit/server/job"
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
	// ptyRequested (terminal_io.go, founder-reported live bug 2026-09-12: "it wont let me type")
	// is set true the moment a real "pty-req" arrives, before "shell" (and therefore before
	// handleConn/runSSHClaimFlow ever read anything) -- read-then-write-once by the single
	// per-channel goroutine in handleSSHChannels, so no lock is needed. Once a PTY is negotiated,
	// a real SSH client disables its OWN local echo and expects the remote side to echo -- a
	// non-PTY (scripted/exec) client never sets this and gets no server-side echo, matching real
	// sshd behavior.
	ptyRequested bool
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
// sshConn is passed through so the "shell" case can read the fingerprint/authorized-key-line
// buildSSHServerConfig's own PublicKeyCallback stashed in its Permissions.Extensions during auth.
func handleSSHChannels(sshConn *ssh.ServerConn, chans <-chan ssh.NewChannel, underlying net.Conn) {
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
					adapter.ptyRequested = true
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
						isGuest, preset := resolveSSHIdentity(sshConn, adapter)
						done := make(chan struct{})
						go sshSessionWatchdog(adapter, done)
						handleConn(adapter, isGuest, preset, adapter.ptyRequested)
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

// resolveSSHIdentity implements §3.1 end to end for one SSH session: known fingerprint -> resume
// that character, no prompt, isGuest=false; unknown fingerprint -> run the claim flow (which
// either returns a freshly bound identity or, on abandonment/failure, nothing); IDUNA unreachable
// -> fall back to guest, same best-effort posture every other idunaclient call in this codebase
// already has, rather than refusing the connection outright over a transient lookup failure.
func resolveSSHIdentity(sshConn *ssh.ServerConn, adapter *sshConnAdapter) (isGuest bool, preset *presetIdentity) {
	var fingerprint, pubKeyLine string
	if sshConn.Permissions != nil {
		fingerprint = sshConn.Permissions.Extensions["fingerprint"]
		pubKeyLine = sshConn.Permissions.Extensions["pubkey-authorized-line"]
	}
	if fingerprint == "" {
		// Should be unreachable (PublicKeyCallback always sets this on the only auth path this
		// server accepts), but a missing fingerprint must never crash a session -- degrade to
		// guest, same as any other identity-resolution failure.
		return true, nil
	}

	characterID, err := gw.iduna.ResolveSSHFingerprint(fingerprint)
	switch {
	case err == nil:
		ch, err := gw.iduna.GetCharacter(characterID)
		if err != nil {
			log.Printf("[ssh] fingerprint resolved to %s but GetCharacter failed, falling back to guest: %v", characterID, err)
			return true, nil
		}
		log.Printf("[ssh] known fingerprint, resuming character=%s name=%s remote=%s", characterID, ch.Name, adapter.RemoteAddr())
		return false, &presetIdentity{name: ch.Name, characterID: characterID, fingerprint: fingerprint}
	case errors.Is(err, idunaclient.ErrNotFound):
		if preset := runSSHClaimFlow(adapter, fingerprint, pubKeyLine); preset != nil {
			return false, preset
		}
		return true, nil
	default:
		log.Printf("[ssh] fingerprint lookup failed, falling back to guest: %v", err)
		return true, nil
	}
}

const (
	sshClaimMaxAttemptsPerIPPerHour = 3 // §3.3 "rate-limit new character claims per source IP per hour"
	sshClaimNameMinLen              = 2
	sshClaimNameMaxLen              = 20
)

// sshReservedNames (§3.3 "reserve an operator namespace before any public announcement") -- a
// real, honest starting list, not exhaustive: this codebase has no broader operator-namespace
// reservation system yet, so these are the specific names most likely to be used for
// impersonation (the game's own name, common staff-role words). Extend as real need appears.
var sshReservedNames = map[string]bool{
	"dragonsnshit":  true,
	"admin":         true,
	"administrator": true,
	"moderator":     true,
	"mod":           true,
	"gm":            true,
	"gamemaster":    true,
	"system":        true,
	"wotan":         true,
	"iduna":         true,
	"support":       true,
}

var (
	sshClaimAttemptsMu sync.Mutex
	sshClaimAttempts   = map[string][]time.Time{}
)

// sshClaimRateLimited enforces §3.3's per-IP-per-hour claim cap, pruning attempts older than an
// hour on every check rather than keeping a separate cleanup goroutine.
func sshClaimRateLimited(ip string) bool {
	sshClaimAttemptsMu.Lock()
	defer sshClaimAttemptsMu.Unlock()
	now := time.Now()
	cutoff := now.Add(-time.Hour)
	kept := sshClaimAttempts[ip][:0]
	for _, t := range sshClaimAttempts[ip] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= sshClaimMaxAttemptsPerIPPerHour {
		sshClaimAttempts[ip] = kept
		return true
	}
	sshClaimAttempts[ip] = append(kept, now)
	return false
}

// validateSSHClaimName applies §3.3's abuse controls to a chosen name: length, a conservative
// charset (letters and digits only -- no impersonation via lookalike punctuation/whitespace), and
// the reserved-name list above. Collision against an already-existing DIFFERENT character's name
// is NOT checked here -- that's enforced by characters.name's own real UNIQUE constraint at the
// CreateCharacter layer (IDUNA's handleCreateCharacter already returns 409, surfaced here as
// idunaclient.ErrConflict), so it isn't duplicated.
func validateSSHClaimName(name string) (ok bool, reason string) {
	if len(name) < sshClaimNameMinLen || len(name) > sshClaimNameMaxLen {
		return false, fmt.Sprintf("Name must be %d-%d characters.", sshClaimNameMinLen, sshClaimNameMaxLen)
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')) {
			return false, "Name must be letters and digits only."
		}
	}
	if sshReservedNames[strings.ToLower(name)] {
		return false, "That name is reserved."
	}
	return true, ""
}

// runSSHClaimFlow implements §3.1's "unknown key -> claim flow -> fingerprint bound to chosen
// name, permanently." Prompts directly over the raw channel (handleConn hasn't taken over yet --
// there is no player/game-world state until a real character exists to attach it to). Returns nil
// (caller falls back to guest) on rate-limit, read error/disconnect, or repeated abandonment --
// never blocks forever.
func runSSHClaimFlow(adapter *sshConnAdapter, fingerprint, pubKeyLine string) *presetIdentity {
	ip := ipOnly(adapter.RemoteAddr())
	send := func(s string) { adapter.Write([]byte(s + "\r\n")) }

	if sshClaimRateLimited(ip) {
		send("Too many new-character attempts from your address recently -- try again later.")
		log.Printf("[ssh-claim] rate-limited ip=%s", ip)
		return nil
	}

	send("No character is bound to this SSH key yet.")
	send("Choose a permanent character name (2-20 letters/digits) -- this key will be bound to it for good.")

	reader := bufio.NewReader(adapter)
	const maxAttempts = 5 // bounded retry loop -- a client stuck retyping an invalid/taken name forever still terminates
	for attempt := 0; attempt < maxAttempts; attempt++ {
		send("Name: ")
		// terminal_io.go: real line editing + CR/LF normalization, and echoes each keystroke
		// back when this session negotiated a real PTY -- a plain ReadString('\n') here is
		// exactly the founder-reported live bug ("it wont let me type"): a real PTY client sends
		// a bare '\r' on Enter (ReadString('\n') never returns) and does its OWN local echo only
		// when there's no PTY, so a PTY session sees nothing it types without this.
		line, err := readTerminalLine(reader, adapter, adapter.ptyRequested)
		if err != nil {
			return nil
		}
		name := strings.TrimSpace(line)
		if ok, reason := validateSSHClaimName(name); !ok {
			send(reason)
			continue
		}

		newID, err := gw.iduna.CreateCharacter(mudPlayerIDFor(name), name, job.WAR)
		switch {
		case err == nil:
			if bindErr := gw.iduna.BindSSHKey(newID, fingerprint, pubKeyLine); bindErr != nil {
				// Real, honest failure mode: the character now exists in IDUNA but the key
				// didn't bind -- logged loudly (not just best-effort-silent) since this leaves a
				// real orphaned character behind, unlike every other best-effort IDUNA call in
				// this file. Falls back to guest for THIS session rather than leaving the player
				// stuck; the character can be claimed properly on a future attempt once IDUNA is
				// reachable again (CreateCharacter's own name-collision handling below covers a
				// retry with the same name failing safely).
				log.Printf("[ssh-claim] character %s created but BindSSHKey failed (orphaned): %v", newID, bindErr)
				send("Character created, but the key binding failed -- reconnect to try again.")
				return nil
			}
			log.Printf("[ssh-claim] new identity claimed character=%s name=%s ip=%s", newID, name, ip)
			send(fmt.Sprintf("Welcome, %s! Your SSH key is now permanently bound to this character.", name))
			return &presetIdentity{name: name, characterID: newID, fingerprint: fingerprint}
		case errors.Is(err, idunaclient.ErrConflict):
			// §3.1 acceptance: "Different key, same claimed name -> refused, not silently
			// reassigned." Re-prompt rather than silently falling back to some other identity.
			send("That name is already taken. Choose another.")
		default:
			send("Could not reach IDUNA to create that character -- try again shortly.")
			return nil
		}
	}
	send("Too many attempts -- disconnecting.")
	return nil
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
			//
			// Stashing the fingerprint + full authorized_keys-line here (Stage 5, §3.1) is the
			// standard x/crypto/ssh pattern for passing auth-time data forward to the session --
			// Permissions survives on the resulting *ssh.ServerConn, read back in
			// resolveSSHIdentity once a "shell" request actually starts a session.
			return &ssh.Permissions{Extensions: map[string]string{
				"fingerprint":            ssh.FingerprintSHA256(key),
				"pubkey-authorized-line": string(ssh.MarshalAuthorizedKey(key)),
			}}, nil
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
		disableNagle(rawConn)

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
			handleSSHChannels(sshConn, chans, rawConn)
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
