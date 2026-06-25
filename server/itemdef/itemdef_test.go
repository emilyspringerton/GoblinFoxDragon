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
