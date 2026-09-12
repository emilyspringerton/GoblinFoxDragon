package main

// terminal_io.go — real, live, founder-reported bug (2026-09-12, over a real Windows OpenSSH
// client, `ssh -p 2222 okemily.com`, right at the SSH claim flow's own "Name:" prompt): "ok i get
// this far and it wont let me type."
//
// Root cause, found by re-checking every raw-input read site in this codebase rather than
// guessing: `runSSHClaimFlow` and both of `handleConn`'s own reads (the guest name prompt, and
// the main per-line command loop) all did a plain `bufio.NewReader(conn).ReadString('\n')`
// straight against the raw connection. That is exactly right for telnet, whose clients (a) do
// their OWN local echo (nothing needs to be echoed back) and (b) send a full "\r\n" on Enter (a
// bare '\n' is present, so ReadString('\n') returns). Neither holds for a real interactive SSH
// client with a PTY allocated: SSH's own real, standard convention is that once a PTY is
// negotiated, the CLIENT disables its own local echo and defers entirely to the remote side (in
// a real remote shell, the remote kernel's own tty driver echoes keystrokes back over the
// channel) -- our server had no such mechanism at all, so every keystroke vanished with zero
// visible feedback. Separately, and independently blocking: a raw PTY's own Enter key sends a
// bare '\r' (CR, 0x0D), not "\r\n" -- `ReadString('\n')` would block forever waiting for a '\n'
// that a real keystroke never sends. (This session's own earlier "live verification" of the SSH
// claim flow used a PTY-driving Python script that wrote a literal "\n" directly, which
// sidestepped both real bugs entirely -- a real gap in that verification, named honestly rather
// than repeated silently here: a scripted byte-string is not the same test as a real keyboard.)
//
// readTerminalLine below is the one, real, shared fix for all three read sites: a real,
// minimal line editor (backspace/DEL actually removes the last buffered byte, not just a raw
// control byte silently smuggled into the name/command string) that treats either a bare '\r' or
// a bare '\n' as ending a line (swallowing a follow-on '\n' right after a '\r', so a real CRLF
// pair from a cooked client is still consumed as exactly one line ending, not two blank lines),
// and optionally echoes each keystroke back to the same connection. `echo` is false for every
// telnet connection (preserves today's exact behavior -- the client already echoes locally, so
// echoing here too would double every character on screen) and true only for an SSH connection
// that actually negotiated a PTY (see sshConnAdapter.ptyRequested) -- a non-interactive/scripted
// SSH client that never sent pty-req gets no server-side echo, matching real ssh/sshd semantics.
//
// Deliberately NOT attempted here: real ANSI/VT escape-sequence handling (arrow keys, Home/End,
// mid-line insertion/deletion) -- an escape sequence's raw bytes get appended into the buffer
// like any other input, a real, honest, minor gap for a MUD-over-a-terminal protocol this simple,
// not a full terminal emulator. maxLineLen defensively caps a client that never sends a line
// terminator at all (accidental or malicious) -- the line is force-completed at the cap rather
// than growing an unbounded buffer.

import (
	"bufio"
	"io"
	"log"
	"net"
)

const maxLineLen = 256

// disableNagle sets TCP_NODELAY on a freshly accepted connection -- a real, live bug this
// session's own echo fix (readTerminalLine, above) exposed: real-time per-keystroke echo writes
// are exactly the small, frequent-write workload Nagle's algorithm (Go's own net.TCPConn default:
// enabled) actively hurts, and combined with the peer's own delayed-ACK timer it produces the
// textbook "Nagle/delayed-ACK death spiral" -- founder-reported live, 2026-09-12: "i have to hit
// enter twice... it wont send until i hit enter or another key[.] as soon as another key is typed
// the previous command sends." Before the echo fix, this server almost never did small, frequent
// writes (one write per full command response, not one per keystroke), so the exact same
// long-standing Nagle-enabled default never surfaced as a felt problem -- a real, latent bug the
// echo fix's own real value (visible per-keystroke feedback) made newly visible, not something
// the echo fix broke on its own. TCP_NODELAY is the standard, correct fix for any interactive
// terminal protocol (a real ssh/telnet server never benefits from Nagle's batching and always
// suffers from its added latency) -- applied to both the SSH listener's raw underlying TCP
// connection (ssh.Channel writes multiplex over it) and the plain telnet listener's own
// connection, matching the exact same real workload once guest telnet also gets per-keystroke
// echo (it doesn't today -- telnet clients echo locally -- but this fix costs nothing to apply
// uniformly and protects against the same class of bug if that ever changes). A non-TCP net.Conn
// (not expected in real use here, but defensive) silently no-ops rather than panicking.
func disableNagle(conn net.Conn) {
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetNoDelay(true); err != nil {
			log.Printf("[net] SetNoDelay failed for %s: %v (interactive latency may suffer, not fatal)", conn.RemoteAddr(), err)
		}
	}
}

// readTerminalLine reads one line of raw user input from r, byte at a time, applying real
// (if minimal) line editing regardless of echo, and writing each processed byte back to w when
// echo is true. Returns the completed line with its line ending stripped (never included), same
// shape callers already got from `strings.TrimSpace(ReadString('\n'))` at every site this
// replaces.
func readTerminalLine(r *bufio.Reader, w io.Writer, echo bool) (string, error) {
	buf := make([]byte, 0, 32)
	for {
		b, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		switch b {
		case '\r':
			if echo {
				io.WriteString(w, "\r\n")
			}
			// A cooked client's own "\r\n" pair must still consume as one line ending, not
			// leave a stray '\n' for the next read to see as an immediate empty line.
			if next, peekErr := r.Peek(1); peekErr == nil && next[0] == '\n' {
				_, _ = r.ReadByte()
			}
			return string(buf), nil
		case '\n':
			if echo {
				io.WriteString(w, "\r\n")
			}
			return string(buf), nil
		case 0x7F, 0x08: // DEL, Backspace
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
				if echo {
					io.WriteString(w, "\b \b") // move back, blank the char, move back again
				}
			}
			continue
		default:
			if len(buf) >= maxLineLen {
				// Force-complete rather than grow unbounded -- a real, defensive cap, not
				// expected to trigger against any real name/command a player actually types.
				return string(buf), nil
			}
			buf = append(buf, b)
			if echo {
				_, _ = w.Write([]byte{b})
			}
			continue
		}
	}
}
