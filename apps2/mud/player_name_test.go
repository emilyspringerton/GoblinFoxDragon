package main

import "testing"

// TestNormalizePlayerName_CaseInsensitive guards the real, founder-reported live gap
// (2026-09-12): "also Emily should be taken too" -- "EMILY", "Emily", and "eMiLy" must all
// normalize to the exact same value.
func TestNormalizePlayerName_CaseInsensitive(t *testing.T) {
	cases := []string{"EMILY", "Emily", "eMiLy", "emily"}
	want := normalizePlayerName(cases[0])
	for _, c := range cases[1:] {
		if got := normalizePlayerName(c); got != want {
			t.Errorf("normalizePlayerName(%q) = %q, want %q (same as normalizePlayerName(%q))", c, got, want, cases[0])
		}
	}
	if want != "emily" {
		t.Errorf("normalizePlayerName(%q) = %q, want %q", cases[0], want, "emily")
	}
}

func TestNormalizePlayerName_DifferentNamesStayDifferent(t *testing.T) {
	if normalizePlayerName("Emily") == normalizePlayerName("Claude") {
		t.Error("two genuinely different names must not normalize to the same value")
	}
}

// TestNameCollidesWithOnlinePlayer_CaseInsensitiveMatch guards the other real half of the same
// founder report: "it should not allow the guest to login as EMILY thats my character on the
// ssh" -- a guest choosing any case variant of an already-online player's name must be caught.
func TestNameCollidesWithOnlinePlayer_CaseInsensitiveMatch(t *testing.T) {
	players := map[string]*player{
		"slot-1": {name: "EMILY"},
	}
	for _, tryName := range []string{"EMILY", "emily", "Emily", "eMiLy"} {
		if !nameCollidesWithOnlinePlayer(players, tryName) {
			t.Errorf("expected %q to collide with online player %q", tryName, "EMILY")
		}
	}
}

func TestNameCollidesWithOnlinePlayer_NoCollisionForADifferentName(t *testing.T) {
	players := map[string]*player{
		"slot-1": {name: "EMILY"},
	}
	if nameCollidesWithOnlinePlayer(players, "SomeoneElse") {
		t.Error("expected no collision for a genuinely different name")
	}
}

func TestNameCollidesWithOnlinePlayer_EmptyPlayersNeverCollides(t *testing.T) {
	if nameCollidesWithOnlinePlayer(map[string]*player{}, "AnyName") {
		t.Error("expected no collision when nobody is online")
	}
}
