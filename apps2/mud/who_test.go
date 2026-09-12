package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	"dragonsnshit/server/homepoint"
)

// TestCmdWho_TagsGuestPlayers guards the real, founder-reported live gap (2026-09-12): "missing
// [guest] tags in who listing" -- an anonymous guest name is unaccountable and can duplicate a
// real, SSH-bound identity's name, so `who` must visibly distinguish the two.
func TestCmdWho_TagsGuestPlayers(t *testing.T) {
	guest := newTestPlayer()
	guest.slot = "guest-slot"
	guest.name = "Rin"
	guest.isGuest = true
	guest.homePoint = homepoint.NewState(0)
	var guestBuf bytes.Buffer
	guest.w = bufio.NewWriter(&guestBuf)

	bound := newTestPlayer()
	bound.slot = "bound-slot"
	bound.name = "Aeon"
	bound.isGuest = false
	bound.homePoint = homepoint.NewState(0)
	var boundBuf bytes.Buffer
	bound.w = bufio.NewWriter(&boundBuf)

	oldGW := gw
	gw = &world{players: map[string]*player{guest.slot: guest, bound.slot: bound}}
	t.Cleanup(func() { gw = oldGW })

	cmdWho(guest)
	out := guestBuf.String()

	var rinLine, aeonLine string
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Rin") {
			rinLine = line
		}
		if strings.Contains(line, "Aeon") {
			aeonLine = line
		}
	}

	if !strings.Contains(rinLine, "[Guest]") {
		t.Errorf("expected the guest's own who-listing line to carry a [Guest] tag, got: %q", rinLine)
	}
	if strings.Contains(aeonLine, "[Guest]") {
		t.Errorf("expected the SSH-bound player's line to NOT carry a [Guest] tag, got: %q", aeonLine)
	}
}
