package main

import (
	"bufio"
	"io"
	"testing"

	combatTp "dragonsnshit/server/combat"
	"dragonsnshit/server/itemdef"
	"dragonsnshit/server/status"
	"dragonsnshit/server/xp"
)

// newTestPlayer returns a minimal *player wired enough to exercise cmdUseItem and its
// dependents. Every field set here is one prompt() itself dereferences (send/sendf/prompt need
// a real, non-nil bufio.Writer; prompt() reads statFX/charXP/tp) -- same "only the fields a
// self-contained function actually touches" shape as weapon_delay_test.go's own
// TestWeaponDelayFor helpers, just a larger set since prompt() (called by every real command
// handler, untested before this) touches more of *player than weaponDelayFor did.
func newTestPlayer() *player {
	return &player{
		w:         bufio.NewWriter(io.Discard),
		statFX:    status.New(),
		charXP:    xp.NewCharXP(),
		tp:        &combatTp.TPState{},
		inventory: make(map[string]int),
		maxHP:     500,
		maxMP:     100,
	}
}

// TestCmdUseItem_HiPotion documents and fixes the founder-reported live bug (2026-09-12):
// "the mud items arent implemented like echo drops and hi potion" -- Hi-Potion used to always
// report "nothing happens yet" regardless of HP.
func TestCmdUseItem_HiPotion_RestoresHP(t *testing.T) {
	itemdefReg = itemdef.NewRegistry()
	if err := itemdefReg.LoadJSON([]byte(`[{"id":302,"name":"Hi-Potion","category":"consumable","stack_size":12}]`)); err != nil {
		t.Fatalf("load test item: %v", err)
	}

	p := newTestPlayer()
	p.hp = 100
	p.inventory["hi-potion"] = 1

	cmdUseItem(p, "hi-potion")

	if p.hp != 200 {
		t.Errorf("hp after Hi-Potion: got %d, want 200 (100 + 100)", p.hp)
	}
	if p.inventory["hi-potion"] != 0 {
		t.Errorf("inventory after use: got %d, want 0 (consumed)", p.inventory["hi-potion"])
	}
}

func TestCmdUseItem_HiPotion_ClampsToMaxHP(t *testing.T) {
	itemdefReg = itemdef.NewRegistry()
	if err := itemdefReg.LoadJSON([]byte(`[{"id":302,"name":"Hi-Potion","category":"consumable","stack_size":12}]`)); err != nil {
		t.Fatalf("load test item: %v", err)
	}

	p := newTestPlayer()
	p.hp = 450 // within 100 of maxHP (500)
	p.inventory["hi-potion"] = 1

	cmdUseItem(p, "hi-potion")

	if p.hp != p.maxHP {
		t.Errorf("hp after Hi-Potion near cap: got %d, want clamped to maxHP %d", p.hp, p.maxHP)
	}
}

func TestCmdUseItem_Ether_RestoresMP(t *testing.T) {
	itemdefReg = itemdef.NewRegistry()
	if err := itemdefReg.LoadJSON([]byte(`[{"id":303,"name":"Ether","category":"consumable","stack_size":12}]`)); err != nil {
		t.Fatalf("load test item: %v", err)
	}

	p := newTestPlayer()
	p.mp = 10
	p.inventory["ether"] = 1

	cmdUseItem(p, "ether")

	if p.mp != 40 {
		t.Errorf("mp after Ether: got %d, want 40 (10 + 30)", p.mp)
	}
	if p.inventory["ether"] != 0 {
		t.Errorf("inventory after use: got %d, want 0 (consumed)", p.inventory["ether"])
	}
}

// TestCmdUseItem_EchoDrop documents the real, literal reason the founder saw Echo Drop as "not
// implemented": it was already fully working (cmdRemoveDebuffs), just unreachable via `use` --
// echo-drop isn't an itemdefReg entry at all, so the old code path fell into "can't be used."
func TestCmdUseItem_EchoDrop_RemovesDebuffsAndConsumesItem(t *testing.T) {
	p := newTestPlayer()
	p.inventory["echo-drop"] = 1
	p.statFX.Apply(status.Effect{Kind: status.Poison, Potency: 1})

	cmdUseItem(p, "echo-drop")

	if p.statFX.Has(status.Poison) {
		t.Error("Poison should be removed after using Echo Drop")
	}
	if p.inventory["echo-drop"] != 0 {
		t.Errorf("inventory after use: got %d, want 0 (consumed)", p.inventory["echo-drop"])
	}
}

func TestCmdUseItem_Antidote_CuresPoisonOnly(t *testing.T) {
	p := newTestPlayer()
	p.inventory["antidote"] = 1
	p.statFX.Apply(status.Effect{Kind: status.Poison, Potency: 1})
	p.statFX.Apply(status.Effect{Kind: status.Slow, Potency: 1})

	cmdUseItem(p, "antidote")

	if p.statFX.Has(status.Poison) {
		t.Error("Poison should be removed after using Antidote")
	}
	if !p.statFX.Has(status.Slow) {
		t.Error("Slow should NOT be removed by Antidote (it only cures Poison)")
	}
	if p.inventory["antidote"] != 0 {
		t.Errorf("inventory after use: got %d, want 0 (consumed)", p.inventory["antidote"])
	}
}

func TestCmdUseItem_NoInventory_DoesNothing(t *testing.T) {
	p := newTestPlayer()
	cmdUseItem(p, "hi-potion")
	if p.hp != 0 {
		t.Errorf("hp should be untouched with no item owned, got %d", p.hp)
	}
}

func TestCmdUseItem_UnknownItem_DoesNotPanic(t *testing.T) {
	itemdefReg = itemdef.NewRegistry()
	p := newTestPlayer()
	p.inventory["mystery-box"] = 1
	cmdUseItem(p, "mystery-box") // must not panic; "can't be used" is the correct, honest reply
}
