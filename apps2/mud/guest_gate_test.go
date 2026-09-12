package main

import (
	"strings"
	"testing"
)

// SSH_TRANSPORT_IDENTITY_SPEC.md Stage 1 (+ Amendment 1): a guest (anonymous telnet) connection
// may never touch economy state or reach another player with free text. These tests verify
// guestGate itself -- the real, server-layer enforcement point handle() calls before its own
// command switch, so a blocked command can never be reached regardless of what a client sends.

func TestGuestGate_BlocksEconomyCommands(t *testing.T) {
	for _, cmd := range []string{"bank", "bazaar", "ah"} {
		p := newTestPlayer()
		p.isGuest = true
		if !guestGate(p, cmd, cmd) {
			t.Errorf("expected guestGate to block %q for a guest", cmd)
		}
	}
}

func TestGuestGate_BlocksChatCommands(t *testing.T) {
	for _, cmd := range []string{"say", "'", "tell", "t", "yell", "y", "guild", "g"} {
		p := newTestPlayer()
		p.isGuest = true
		if !guestGate(p, cmd, cmd+" hello") {
			t.Errorf("expected guestGate to block %q for a guest", cmd)
		}
	}
}

func TestGuestGate_BlocksPartyChatShortcut(t *testing.T) {
	p := newTestPlayer()
	p.isGuest = true
	if !guestGate(p, "/p", "/p hey team") {
		t.Error("expected guestGate to block the /p party-chat shortcut for a guest")
	}
}

func TestGuestGate_AllowsPermittedCommandsForGuests(t *testing.T) {
	// Amendment 1 §B.4: movement, combat, jobs, parties/linkshells (membership, not chat),
	// quests, etc. all remain permitted for a guest.
	for _, cmd := range []string{"look", "n", "attack", "setjob", "party", "invite", "accept", "quests", "ls-create"} {
		p := newTestPlayer()
		p.isGuest = true
		if guestGate(p, cmd, cmd) {
			t.Errorf("expected guestGate to allow %q for a guest (permitted per Amendment 1 §B.4)", cmd)
		}
	}
}

func TestGuestGate_NeverBlocksNonGuestSessions(t *testing.T) {
	// A headless (Town GUI) session already carries a real WOTAN-authenticated identity --
	// isGuest is false for it (Go's own zero value), and none of these commands should ever be
	// blocked for a non-guest.
	for _, cmd := range []string{"bank", "bazaar", "ah", "say", "tell", "yell", "guild"} {
		p := newTestPlayer()
		p.isGuest = false
		if guestGate(p, cmd, cmd) {
			t.Errorf("expected guestGate to never block %q for a non-guest session", cmd)
		}
	}
}

func TestGuestGate_MessageExplainsUpgradeNotBareDenial(t *testing.T) {
	// Amendment 1 §D: "Blocked-command responses explain the upgrade rather than only denying."
	msg := guestBlockedMessage("economy")
	if !strings.Contains(strings.ToLower(msg), "ssh") {
		t.Errorf("expected the economy block message to mention SSH (the real upgrade path), got: %s", msg)
	}
	chatMsg := guestBlockedMessage("chat")
	if !strings.Contains(strings.ToLower(chatMsg), "ssh") {
		t.Errorf("expected the chat block message to mention SSH (the real upgrade path), got: %s", chatMsg)
	}
}

// TestGuestGate_MessageGivesTheRealConnectCommandNotComingSoon guards the real, founder-reported
// live gap (2026-09-12): SSH has been live since Stage 4/5 (2026-09-12), but every guest-facing
// message still said "coming soon" -- stale the moment it shipped. Every guest-blocked message
// must now name the actual, concrete command (not a vague "coming soon"), and none may regress
// back to that wording.
func TestGuestGate_MessageGivesTheRealConnectCommandNotComingSoon(t *testing.T) {
	for _, category := range []string{"economy", "chat", "unknown-category"} {
		msg := guestBlockedMessage(category)
		lower := strings.ToLower(msg)
		if strings.Contains(lower, "coming soon") {
			t.Errorf("guestBlockedMessage(%q) still says \"coming soon\" -- SSH is live, this must name the real command: %s", category, msg)
		}
		if !strings.Contains(msg, "ssh -p 2222 okemily.com") {
			t.Errorf("guestBlockedMessage(%q) should give the real, concrete connect command (ssh -p 2222 okemily.com), got: %s", category, msg)
		}
	}
}
