package main

// SSH_TRANSPORT_IDENTITY_SPEC.md §2/§3 / Stages 4-5 tests. These cover everything that doesn't
// need the game world (`gw`) initialized: host key persistence, per-IP address parsing, the real
// §2.1 auth semantics (password refused, public-key/TOFU accepted, Permissions.Extensions
// correctly carries the fingerprint/authorized-key-line forward for Stage 5), and §3.3's pure
// claim-flow logic (name validation, per-IP rate limiting) -- exercised via a genuine
// ssh.NewServerConn/ssh.NewClientConn handshake over a real loopback TCP connection where a real
// handshake matters, and directly otherwise. resolveSSHIdentity/runSSHClaimFlow themselves need
// `gw` fully initialized (world, zones, a real IDUNA client) -- exactly like
// guest_gate_test.go/use_item_test.go, this package's own existing tests never construct that, so
// end-to-end shell-session behavior (guest banner over SSH, PTY/no-PTY, resize, per-IP session
// cap, and now identity resolution/claim) is live-verified against a real running binary instead,
// matching how every earlier stage was verified in this same codebase.

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestLoadOrGenerateSSHHostKey_GeneratesThenReusesSameKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "host_key")

	first, err := loadOrGenerateSSHHostKey(path)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected key file to be persisted: %v", err)
	}

	second, err := loadOrGenerateSSHHostKey(path)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}

	if string(first.PublicKey().Marshal()) != string(second.PublicKey().Marshal()) {
		t.Fatal("loadOrGenerateSSHHostKey did not reuse the persisted key on second load -- §2.2 requires the host key survive restart")
	}
}

func TestLoadOrGenerateSSHHostKey_PersistedFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "host_key")
	if _, err := loadOrGenerateSSHHostKey(path); err != nil {
		t.Fatalf("load: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("host key file permissions = %o, want 0600 (private key material)", perm)
	}
}

func TestIPOnly(t *testing.T) {
	cases := map[string]string{
		"127.0.0.1:2222":    "127.0.0.1",
		"[::1]:2222":        "::1",
		"not-a-host-port":   "not-a-host-port",
		"203.0.113.7:54321": "203.0.113.7",
	}
	for input, want := range cases {
		got := ipOnly(stubAddr(input))
		if got != want {
			t.Errorf("ipOnly(%q) = %q, want %q", input, got, want)
		}
	}
}

type stubAddr string

func (s stubAddr) Network() string { return "tcp" }
func (s stubAddr) String() string  { return string(s) }

// sshPipeHandshake runs a real ssh.NewServerConn against a real ssh.Dial handshake over a real
// loopback TCP connection -- no `gw`, no session channel ever opened (nothing here sends a
// "shell" request), just the §2.1 auth exchange itself. A real TCP loopback pair is used rather
// than net.Pipe(): net.Pipe() is fully unbuffered/synchronous, and the SSH version-exchange step
// has both sides write their version string before either reads (confirmed live -- an earlier
// net.Pipe()-based version of this helper deadlocked both goroutines in exactly that write).
// Returns the client error (nil on success) so each test can assert on exactly what §2.1
// requires.
func sshPipeHandshake(t *testing.T, clientConfig *ssh.ClientConfig) error {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	hostKey, err := loadOrGenerateSSHHostKey(filepath.Join(t.TempDir(), "host_key"))
	if err != nil {
		t.Fatalf("host key: %v", err)
	}
	config := buildSSHServerConfig(hostKey)

	serverDone := make(chan error, 1)
	go func() {
		serverSide, err := ln.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		_, _, _, err = ssh.NewServerConn(serverSide, config)
		serverDone <- err
	}()

	clientSide, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_, _, _, clientErr := ssh.NewClientConn(clientSide, ln.Addr().String(), clientConfig)

	select {
	case <-serverDone:
	case <-time.After(5 * time.Second):
		t.Fatal("server-side handshake never completed")
	}
	return clientErr
}

func TestSSHAuth_PasswordRefused(t *testing.T) {
	clientConfig := &ssh.ClientConfig{
		User:            "anyone",
		Auth:            []ssh.AuthMethod{ssh.Password("irrelevant")},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if err := sshPipeHandshake(t, clientConfig); err == nil {
		t.Fatal("password auth succeeded -- §2.1 requires password auth be refused (PasswordCallback must be nil)")
	}
}

func TestSSHAuth_KeyboardInteractiveRefused(t *testing.T) {
	clientConfig := &ssh.ClientConfig{
		User: "anyone",
		Auth: []ssh.AuthMethod{ssh.KeyboardInteractive(func(name, instruction string, questions []string, echos []bool) ([]string, error) {
			return nil, nil
		})},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if err := sshPipeHandshake(t, clientConfig); err == nil {
		t.Fatal("keyboard-interactive auth succeeded -- §2.1 requires it be refused (KeyboardInteractiveCallback must be nil)")
	}
}

func TestSSHAuth_AnyPublicKeyAcceptedTOFU(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}

	clientConfig := &ssh.ClientConfig{
		User:            "brand-new-never-seen-before",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if err := sshPipeHandshake(t, clientConfig); err != nil {
		t.Fatalf("a fresh, never-registered public key was refused -- §2.1 requires trust-on-first-use (any offered key accepted for the connection): %v", err)
	}
}

func TestSSHAuth_DifferentUnregisteredKeysBothAcceptedTOFU(t *testing.T) {
	// Real TOFU semantics check: TWO different, unrelated keys, neither ever seen before, both
	// succeed -- proves the server isn't accidentally binding/remembering the first key it saw
	// (that would be Stage 5's job, not Stage 4's).
	for i := 0; i < 2; i++ {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatalf("generate client key %d: %v", i, err)
		}
		signer, err := ssh.NewSignerFromKey(priv)
		if err != nil {
			t.Fatalf("signer %d: %v", i, err)
		}
		clientConfig := &ssh.ClientConfig{
			User:            "guest",
			Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}
		if err := sshPipeHandshake(t, clientConfig); err != nil {
			t.Fatalf("key %d refused: %v", i, err)
		}
	}
}

func TestSSHAuth_PermissionsCarryFingerprintAndAuthorizedLine(t *testing.T) {
	// Stage 5 (§3.1): resolveSSHIdentity reads sshConn.Permissions.Extensions after auth --
	// verified here directly against a real handshake rather than trusting buildSSHServerConfig's
	// own PublicKeyCallback in isolation, since a typo in the Extensions map key would silently
	// break identity resolution without ever failing the handshake itself.
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate client key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	wantFingerprint := ssh.FingerprintSHA256(signer.PublicKey())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	hostKey, err := loadOrGenerateSSHHostKey(filepath.Join(t.TempDir(), "host_key"))
	if err != nil {
		t.Fatalf("host key: %v", err)
	}
	config := buildSSHServerConfig(hostKey)

	serverPerms := make(chan *ssh.Permissions, 1)
	go func() {
		rawConn, err := ln.Accept()
		if err != nil {
			serverPerms <- nil
			return
		}
		sshConn, _, _, err := ssh.NewServerConn(rawConn, config)
		if err != nil {
			serverPerms <- nil
			return
		}
		serverPerms <- sshConn.Permissions
	}()

	clientConn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	clientConfig := &ssh.ClientConfig{
		User:            "test",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if _, _, _, err := ssh.NewClientConn(clientConn, ln.Addr().String(), clientConfig); err != nil {
		t.Fatalf("client handshake: %v", err)
	}

	perms := <-serverPerms
	if perms == nil {
		t.Fatal("server-side handshake failed")
	}
	if perms.Extensions["fingerprint"] != wantFingerprint {
		t.Errorf("Permissions.Extensions[\"fingerprint\"] = %q, want %q", perms.Extensions["fingerprint"], wantFingerprint)
	}
	if !strings.HasPrefix(perms.Extensions["pubkey-authorized-line"], "ssh-ed25519 ") {
		t.Errorf("Permissions.Extensions[\"pubkey-authorized-line\"] = %q, want it to start with \"ssh-ed25519 \"", perms.Extensions["pubkey-authorized-line"])
	}
}

func TestValidateSSHClaimName(t *testing.T) {
	cases := map[string]bool{
		"Bob":                   true,
		"Xx":                    true,  // exactly the 2-char minimum
		"a23456789012345678901": false, // 21 chars, over the max
		"X":                     false, // 1 char, under the min
		"Bob Smith":             false, // space
		"Bob-Smith":             false, // hyphen
		"Bob'sChar":             false, // apostrophe
		"admin":                 false, // reserved (exact)
		"Admin":                 false, // reserved, case-insensitive
		"ADMINISTRATOR":         false,
		"GM":                    false,
		"DragonsNShit":          false,
		"Wotan":                 false,
		"NotReserved":           true,
	}
	for name, wantOK := range cases {
		ok, reason := validateSSHClaimName(name)
		if ok != wantOK {
			t.Errorf("validateSSHClaimName(%q) = (%v, %q), want ok=%v", name, ok, reason, wantOK)
		}
		if !ok && reason == "" {
			t.Errorf("validateSSHClaimName(%q) rejected with no reason given to the player", name)
		}
	}
}

func TestSSHClaimRateLimited(t *testing.T) {
	// Isolate from any other test/real traffic touching the shared map.
	sshClaimAttemptsMu.Lock()
	sshClaimAttempts = map[string][]time.Time{}
	sshClaimAttemptsMu.Unlock()

	ip := "203.0.113.55"
	for i := 0; i < sshClaimMaxAttemptsPerIPPerHour; i++ {
		if sshClaimRateLimited(ip) {
			t.Fatalf("attempt %d: rate-limited before reaching the real cap (%d)", i+1, sshClaimMaxAttemptsPerIPPerHour)
		}
	}
	if !sshClaimRateLimited(ip) {
		t.Fatalf("attempt %d: expected rate-limited after %d attempts within the hour", sshClaimMaxAttemptsPerIPPerHour+1, sshClaimMaxAttemptsPerIPPerHour)
	}

	// A different source IP must have its own independent budget.
	if sshClaimRateLimited("203.0.113.99") {
		t.Fatal("a different IP was rate-limited by another IP's own attempts")
	}
}

func TestSSHClaimRateLimited_OldAttemptsExpire(t *testing.T) {
	sshClaimAttemptsMu.Lock()
	ip := "203.0.113.77"
	sshClaimAttempts[ip] = []time.Time{
		time.Now().Add(-2 * time.Hour), // outside the 1-hour window -- must not count
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-2 * time.Hour),
	}
	sshClaimAttemptsMu.Unlock()

	if sshClaimRateLimited(ip) {
		t.Fatal("expired attempts outside the 1-hour window should not count toward the cap")
	}
}
