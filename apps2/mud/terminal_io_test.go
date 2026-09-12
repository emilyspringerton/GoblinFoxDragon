package main

import (
	"bufio"
	"bytes"
	"net"
	"strings"
	"testing"
)

// TestReadTerminalLine_BareCRTerminatesLine guards the real root cause of the founder-reported
// live bug (2026-09-12, real Windows OpenSSH client): a raw PTY's own Enter key sends a bare '\r'
// (0x0D), not "\r\n" -- the old `bufio.Reader.ReadString('\n')` this replaced would block forever
// waiting for a '\n' that keystroke never sends.
func TestReadTerminalLine_BareCRTerminatesLine(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("CoolName1\r"))
	var out bytes.Buffer
	line, err := readTerminalLine(r, &out, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "CoolName1" {
		t.Errorf("readTerminalLine on bare CR = %q, want %q", line, "CoolName1")
	}
}

func TestReadTerminalLine_CRLFPairConsumedAsOneLineEnding(t *testing.T) {
	// A cooked telnet client's own "\r\n" must not leave a stray '\n' that a second read would
	// see as an immediate empty line.
	r := bufio.NewReader(strings.NewReader("first\r\nsecond\r\n"))
	var out bytes.Buffer
	line1, err := readTerminalLine(r, &out, false)
	if err != nil || line1 != "first" {
		t.Fatalf("first line = %q, err=%v, want %q, nil", line1, err, "first")
	}
	line2, err := readTerminalLine(r, &out, false)
	if err != nil || line2 != "second" {
		t.Fatalf("second line = %q, err=%v, want %q, nil (a stray swallowed '\\n' would give an empty first line here)", line2, err, "second")
	}
}

func TestReadTerminalLine_BareLFAlsoTerminatesLine(t *testing.T) {
	// A raw netcat-style client (or this session's own earlier PTY-driving test scripts) sending
	// a lone '\n' must keep working exactly as it always has.
	r := bufio.NewReader(strings.NewReader("plainname\n"))
	var out bytes.Buffer
	line, err := readTerminalLine(r, &out, false)
	if err != nil || line != "plainname" {
		t.Fatalf("readTerminalLine on bare LF = %q, err=%v, want %q, nil", line, err, "plainname")
	}
}

// TestReadTerminalLine_EchoesKeystrokesWhenEnabled guards the OTHER real half of the founder-
// reported bug: nothing was ever echoed back to a real PTY session, so a player saw nothing they
// typed even though the byte-blocking issue above were somehow not present.
func TestReadTerminalLine_EchoesKeystrokesWhenEnabled(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("Rin\r"))
	var out bytes.Buffer
	line, err := readTerminalLine(r, &out, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "Rin" {
		t.Fatalf("line = %q, want %q", line, "Rin")
	}
	if got := out.String(); got != "Rin\r\n" {
		t.Errorf("echoed output = %q, want %q (each keystroke echoed, CR echoed as a real CRLF)", got, "Rin\r\n")
	}
}

func TestReadTerminalLine_NoEchoWritesNothingBack(t *testing.T) {
	// Telnet's own real path (echo=false): the client already echoes locally, so this server
	// must write NOTHING back, or every character would double on screen.
	r := bufio.NewReader(strings.NewReader("Rin\r"))
	var out bytes.Buffer
	if _, err := readTerminalLine(r, &out, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("echo=false must write nothing back to the connection, got %q", out.String())
	}
}

// TestReadTerminalLine_BackspaceActuallyRemovesFromBuffer is the real, meaningful half of "line
// editing" -- a naive transport-level echo alone cannot fix this: ReadString('\n') would have
// smuggled the raw backspace byte itself into the resulting string, corrupting it, not erased the
// intended character.
func TestReadTerminalLine_BackspaceActuallyRemovesFromBuffer(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"DEL 0x7F", "Rinn\x7F\r"},                // typed "Rinn", backspaced once -> "Rin"
		{"BS 0x08", "Rinn\x08\r"},                 // some clients send BS instead of DEL
		{"backspace past start", "\x7F\x7FRin\r"}, // backspacing with an empty buffer must not panic or underflow
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := bufio.NewReader(strings.NewReader(c.input))
			var out bytes.Buffer
			line, err := readTerminalLine(r, &out, false)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if line != "Rin" {
				t.Errorf("readTerminalLine(%q) = %q, want %q", c.input, line, "Rin")
			}
		})
	}
}

func TestReadTerminalLine_BackspaceEchoesEraseSequence(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("Rx\x7Fn\r"))
	var out bytes.Buffer
	line, err := readTerminalLine(r, &out, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if line != "Rn" {
		t.Fatalf("line = %q, want %q", line, "Rn")
	}
	want := "R" + "x" + "\b \b" + "n" + "\r\n"
	if got := out.String(); got != want {
		t.Errorf("echoed output = %q, want %q", got, want)
	}
}

func TestReadTerminalLine_EOFPropagatesAsError(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("no-line-ending-at-all"))
	var out bytes.Buffer
	if _, err := readTerminalLine(r, &out, false); err == nil {
		t.Error("expected an error when the connection closes mid-line with no terminator")
	}
}

// TestDisableNagle_RealTCPConnDoesNotError guards the real, founder-reported live bug
// (2026-09-12): "i have to hit enter twice... it wont send until i hit enter or another key" --
// Nagle's algorithm (Go's own net.TCPConn default) batching this file's own new per-keystroke
// echo writes. A real loopback TCP pair (not a mock) confirms SetNoDelay actually succeeds
// against a real *net.TCPConn.
func TestDisableNagle_RealTCPConnDoesNotError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	acceptedCh := make(chan net.Conn, 1)
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			acceptedCh <- conn
		}
	}()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	server := <-acceptedCh
	defer server.Close()

	// Must not panic and must actually be a *net.TCPConn this real dial/accept pair.
	if _, ok := server.(*net.TCPConn); !ok {
		t.Fatalf("expected a real *net.TCPConn from a real TCP accept, got %T", server)
	}
	disableNagle(server) // real assertion: this must not panic or otherwise misbehave
}

// TestDisableNagle_NonTCPConnNoOps guards the defensive fallback -- a net.Pipe() conn (used
// elsewhere in this package's own tests) is not a *net.TCPConn, and must silently no-op rather
// than panicking on a failed type assertion.
func TestDisableNagle_NonTCPConnNoOps(t *testing.T) {
	clientSide, serverSide := net.Pipe()
	defer clientSide.Close()
	defer serverSide.Close()
	disableNagle(serverSide) // must not panic
}

func TestReadTerminalLine_CapsUnboundedLineAtMaxLen(t *testing.T) {
	huge := strings.Repeat("a", maxLineLen+50) + "\r"
	r := bufio.NewReader(strings.NewReader(huge))
	var out bytes.Buffer
	line, err := readTerminalLine(r, &out, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(line) > maxLineLen {
		t.Errorf("line length = %d, want capped at %d", len(line), maxLineLen)
	}
}
