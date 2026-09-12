package main

// SSH_TRANSPORT_IDENTITY_SPEC.md §2 / Stage 4 tests. These cover everything that doesn't need
// the game world (`gw`) initialized: host key persistence, per-IP address parsing, and the real
// §2.1 auth semantics (password refused, public-key/TOFU accepted) exercised via a genuine
// ssh.NewServerConn/ssh.NewClientConn handshake over a real loopback TCP connection. A "shell" request
// would open a real session channel into handleConn, which needs `gw` fully initialized (world,
// zones, IDUNA client) -- exactly like guest_gate_test.go/use_item_test.go, this package's own
// existing tests never construct that, so end-to-end shell-session behavior (guest banner over
// SSH, PTY/no-PTY, resize, per-IP session cap) is live-verified against a real running binary
// instead, matching how Stage 1/2/3 were verified in this same codebase.

import (
	"crypto/ed25519"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
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
