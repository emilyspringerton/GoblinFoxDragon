package burrowgen

import "testing"

// Real proof the generated BURROW-compiled function matches PARENA/stdlib/gfd/nm_bonus_mod.prn's
// own real decision logic (GFD-x-123/124) -- same "prove the generated code directly" convention
// GFD-MACRO-0012's own action_bar_mod tests already established for the C target.
func TestOnGfdMobDeathXpBonusPercent(t *testing.T) {
	if got := OnGfdMobDeathXpBonusPercent(0); got != 0 {
		t.Fatalf("expected 0%% bonus for a non-NM kill, got %d", got)
	}
	if got := OnGfdMobDeathXpBonusPercent(1); got != 50 {
		t.Fatalf("expected 50%% bonus for an NM kill, got %d", got)
	}
}
