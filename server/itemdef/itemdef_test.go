package itemdef

import (
	"encoding/json"
	"testing"
)

var seedJSON = []byte(`[
  {
    "id": 1,
    "name": "Iron Sword",
    "category": "weapon",
    "equip_slots": ["main", "off"],
    "jobs": ["WAR","PLD","DRK","RDM","NIN"],
    "level": 1,
    "stats": {"attack": 10, "str": 2},
    "stack_size": 1,
    "model_id": "sword_iron"
  },
  {
    "id": 2,
    "name": "Leather Armor",
    "category": "armor",
    "equip_slots": ["body"],
    "jobs": ["WAR","MNK","THF","BST","NIN","DNC"],
    "level": 4,
    "stats": {"defense": 8, "vit": 1},
    "stack_size": 1
  },
  {
    "id": 3,
    "name": "Chocobo Egg",
    "category": "key_item",
    "stack_size": 1,
    "flags": ["rare","ex"]
  },
  {
    "id": 4,
    "name": "Fire Crystal",
    "category": "crystal",
    "stack_size": 12,
    "stats": {}
  },
  {
    "id": 5,
    "name": "Excalibur",
    "category": "weapon",
    "equip_slots": ["main"],
    "jobs": ["PLD"],
    "level": 73,
    "stats": {"attack": 249, "str": 25, "accuracy": 20},
    "stack_size": 1,
    "flags": ["rare","ex"]
  }
]`)

func TestRegistryLoad(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON(seedJSON); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if r.Len() != 5 {
		t.Errorf("Len=%d want 5", r.Len())
	}
}

func TestByID(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, ok := r.ByID(1)
	if !ok {
		t.Fatal("ByID(1) not found")
	}
	if d.Name != "Iron Sword" {
		t.Errorf("name=%q want Iron Sword", d.Name)
	}
}

func TestByName(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, ok := r.ByName("excalibur")
	if !ok {
		t.Fatal("ByName(excalibur) not found")
	}
	if d.ID != 5 {
		t.Errorf("id=%d want 5", d.ID)
	}
}

// TestByName_HyphenatedMultiWordName -- S251-09's real, found-live bug: a
// multi-word item name registered under its own literal, spaced, lowercased
// form ("fire crystal"), but every real shop-facing call site in
// apps2/mud/main.go passes the hyphenated convention ("fire-crystal")
// instead, so no multi-word item was ever reachable by its real shop id.
func TestByName_HyphenatedMultiWordName(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, ok := r.ByName("fire-crystal")
	if !ok {
		t.Fatal("ByName(fire-crystal) not found -- the real shop-facing id must resolve")
	}
	if d.ID != 4 {
		t.Errorf("id=%d want 4", d.ID)
	}
	// The un-hyphenated, spaced form (and mixed case) must keep working too --
	// this fix normalizes both sides of the lookup, not just one.
	d2, ok := r.ByName("Fire Crystal")
	if !ok || d2.ID != 4 {
		t.Fatalf("ByName(Fire Crystal) = %v, %v -- want id 4, ok", d2, ok)
	}
}

func TestJobMask(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, _ := r.ByID(1) // Iron Sword: WAR,PLD,DRK,RDM,NIN

	if !d.JobMask().CanEquipJob("WAR") {
		t.Error("WAR should be able to equip Iron Sword")
	}
	if d.JobMask().CanEquipJob("WHM") {
		t.Error("WHM should NOT be able to equip Iron Sword")
	}
}

func TestAllJobsMask(t *testing.T) {
	m := JobMaskFor(nil) // empty = 0
	// mask 0 = all jobs
	if !m.CanEquipJob("WHM") {
		t.Error("mask 0 should allow WHM")
	}
}

func TestCanEquip_Success(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, _ := r.ByID(1)
	if err := d.CanEquip("main", "WAR", 10); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCanEquip_WrongSlot(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, _ := r.ByID(1)
	if err := d.CanEquip("head", "WAR", 10); err != ErrSlotMismatch {
		t.Errorf("expected ErrSlotMismatch, got %v", err)
	}
}

func TestCanEquip_LevelTooLow(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, _ := r.ByID(5) // Excalibur level 73
	if err := d.CanEquip("main", "PLD", 50); err == nil {
		t.Error("expected level error")
	}
}

func TestCanEquip_JobRestricted(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, _ := r.ByID(5) // Excalibur PLD only
	if err := d.CanEquip("main", "WAR", 75); err == nil {
		t.Error("expected job restriction error")
	}
}

func TestCanEquip_NotEquipment(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, _ := r.ByID(3) // Chocobo Egg — key item
	if err := d.CanEquip("main", "WAR", 10); err != ErrNotEquipment {
		t.Errorf("expected ErrNotEquipment, got %v", err)
	}
}

func TestFlags(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	d, _ := r.ByID(3) // Chocobo Egg — rare+ex
	if !d.Flags().Has(FlagRare) {
		t.Error("expected FlagRare")
	}
	if !d.Flags().Has(FlagEx) {
		t.Error("expected FlagEx")
	}
	iron, _ := r.ByID(1)
	if iron.Flags().Has(FlagRare) {
		t.Error("Iron Sword should not be rare")
	}
}

func TestStackSize(t *testing.T) {
	r := NewRegistry()
	r.LoadJSON(seedJSON)
	crystal, _ := r.ByID(4)
	if crystal.StackSize != 12 {
		t.Errorf("crystal stack_size=%d want 12", crystal.StackSize)
	}
	sword, _ := r.ByID(1)
	if sword.StackSize != 1 {
		t.Errorf("sword stack_size=%d want 1", sword.StackSize)
	}
}

func TestInvalidJSON(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte("not json")); err == nil {
		t.Error("expected parse error")
	}
}

func TestMarshalRoundtrip(t *testing.T) {
	// Verify that items.json can roundtrip through JSON encode/decode.
	var defs []ItemDef
	if err := json.Unmarshal(seedJSON, &defs); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	out, err := json.Marshal(defs)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var defs2 []ItemDef
	if err := json.Unmarshal(out, &defs2); err != nil {
		t.Fatalf("second unmarshal: %v", err)
	}
	if len(defs2) != len(defs) {
		t.Errorf("roundtrip len %d → %d", len(defs), len(defs2))
	}
}

// TestLoadJSON_NormalizesInconsistentStatKeyCasing guards the real, found-live bug (2026-09-12,
// founder scoping a starting-weapon feature): data/items.json's own real items use inconsistent
// stat-key casing ("STR" alongside "str", "Attack" alongside "attack", "Magic Attack Bonus" with
// literal spaces alongside "magic_attack_bonus"). gear.Equipment.ComputeStats sums by raw string
// key with zero normalization of its own, so two items differing only in key casing would land
// in separate map keys and silently not combine -- this must be fixed once, at load time, not
// left for every future reader of Stats to rediscover.
func TestLoadJSON_NormalizesInconsistentStatKeyCasing(t *testing.T) {
	r := NewRegistry()
	raw := []byte(`[
		{"id": 900, "name": "Upper Item", "category": "weapon", "stats": {"STR": 5, "Attack": 10}},
		{"id": 901, "name": "Lower Item", "category": "weapon", "stats": {"str": 3, "attack": 7}},
		{"id": 902, "name": "Spaced Item", "category": "weapon", "stats": {"Magic Attack Bonus": 12}}
	]`)
	if err := r.LoadJSON(raw); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	upper, _ := r.ByID(900)
	if upper.Stats["str"] != 5 || upper.Stats["attack"] != 10 {
		t.Errorf("uppercase-keyed item did not normalize: %+v", upper.Stats)
	}
	if _, hasRawUpper := upper.Stats["STR"]; hasRawUpper {
		t.Error("raw uppercase key 'STR' should not survive normalization")
	}
	lower, _ := r.ByID(901)
	if lower.Stats["str"] != 3 || lower.Stats["attack"] != 7 {
		t.Errorf("lowercase-keyed item changed unexpectedly: %+v", lower.Stats)
	}
	spaced, _ := r.ByID(902)
	if spaced.Stats["magic_attack_bonus"] != 12 {
		t.Errorf("spaced key did not normalize to underscored form: %+v", spaced.Stats)
	}
}

func TestNormalizeStatKeys_NilMapStaysNil(t *testing.T) {
	if got := normalizeStatKeys(nil); got != nil {
		t.Errorf("normalizeStatKeys(nil) = %v, want nil", got)
	}
}

func TestNormalizeStatKeys_CollidingKeysCombineRatherThanOverwrite(t *testing.T) {
	// A real, defensive case: two raw keys that normalize to the same real stat (not expected in
	// today's data, but a real possibility for hand-authored JSON) must sum, not silently drop one.
	got := normalizeStatKeys(map[string]int{"STR": 5, "str": 3})
	if got["str"] != 8 {
		t.Errorf("normalizeStatKeys colliding keys = %v, want str=8 (5+3)", got)
	}
}
