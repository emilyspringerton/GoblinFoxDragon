package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"

	combatTp "dragonsnshit/server/combat"
	"dragonsnshit/server/gear"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/xp"
)

// cmdws_gate_test.go guards S412-06's real new gate on cmdWS: before this pass, `setws`/`ws`
// accepted any canonical weapon skill name with zero check against what's actually equipped or
// how skilled the player really is (BACKLOG S412-06: "setws currently accepts any name, no
// gate"). Both real refusal paths return before touching `gw`/the mob registry at all, so these
// tests don't need a full world/mob setup -- a real target ID plus full TP is enough to reach the
// gate; prompt()/zoneName() are already nil-safe against a missing `gw` (use_item_test.go's own
// established precedent for testing command handlers without a real world).

func newWSGateTestPlayer(t *testing.T) (*player, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	p := &player{
		w:      bufio.NewWriter(&buf),
		statFX: newTestPlayer().statFX,
		charXP: xp.NewCharXP(),
		tp:     &combatTp.TPState{Current: 100},
		combat: &mob.PlayerCombat{TargetMobID: "test-mob-1"},
		equip:  gear.NewEquipment(),
	}
	return p, &buf
}

func TestCmdWS_RefusesWrongWeaponTypeEquipped(t *testing.T) {
	p, buf := newWSGateTestPlayer(t)
	p.wsSkill = "Fast Blade" // sword WS -- player is unarmed (h2h), not sword

	cmdWS(p, "")

	out := buf.String()
	if !strings.Contains(out, "requires a sword equipped") || !strings.Contains(out, "h2h") {
		t.Errorf("expected a real weapon-type refusal message, got: %q", out)
	}
	if p.tp.Current != 100 {
		t.Errorf("TP should NOT be consumed on a gate refusal, got %d", p.tp.Current)
	}
}

func TestCmdWS_RefusesInsufficientSkillEvenWithRightWeaponType(t *testing.T) {
	oldReg := itemdefReg
	itemdefReg = testRegistryWithTypedSword(t)
	t.Cleanup(func() { itemdefReg = oldReg })

	p, buf := newWSGateTestPlayer(t)
	if err := p.equip.Equip(gear.SlotMainHand, gear.ItemEntry{ItemID: "sword", DefID: 3}); err != nil {
		t.Fatalf("equip: %v", err)
	}
	p.wsSkill = "Red Lotus Blade" // sword WS, MinSkillLevel 85 -- a fresh player has 0 sword skill

	cmdWS(p, "")

	out := buf.String()
	if !strings.Contains(out, "Not skilled enough") || !strings.Contains(out, "sword") {
		t.Errorf("expected a real insufficient-skill refusal message, got: %q", out)
	}
	if p.tp.Current != 100 {
		t.Errorf("TP should NOT be consumed on a gate refusal, got %d", p.tp.Current)
	}
}

func TestCmdWS_UnknownNameRefusalUnaffectedByGate(t *testing.T) {
	p, buf := newWSGateTestPlayer(t)
	p.wsSkill = "Totally Made Up Skill Name"

	cmdWS(p, "")

	out := buf.String()
	if !strings.Contains(out, "Unknown weapon skill") {
		t.Errorf("expected the pre-existing unknown-skill message to still fire, got: %q", out)
	}
}
