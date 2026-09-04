package mobdrop

import "testing"

func TestDropsFor_RegisteredKind_ReturnsItsOwnTable(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[
		{"kind":"worm","items":[{"id":"worm-sinew","name":"Worm Sinew"},{"id":"earth-crystal","name":"Earth Crystal"}]}
	]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	drops := r.DropsFor("worm")
	if len(drops) != 2 || drops[0].ID != "worm-sinew" || drops[1].ID != "earth-crystal" {
		t.Fatalf("unexpected drops: %+v", drops)
	}
}

func TestDropsFor_CaseInsensitive(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"kind":"King Worm","items":[{"id":"king-sinew","name":"King Worm Sinew"}]}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	drops := r.DropsFor("king worm")
	if len(drops) != 1 || drops[0].ID != "king-sinew" {
		t.Fatalf("expected case-insensitive match, got: %+v", drops)
	}
}

func TestDropsFor_UnregisteredKind_FallsBackToDefaultDrop(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"kind":"worm","items":[{"id":"worm-sinew","name":"Worm Sinew"}]}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	drops := r.DropsFor("cave-bat")
	if len(drops) != 1 || drops[0] != DefaultDrop {
		t.Fatalf("expected DefaultDrop fallback, got: %+v", drops)
	}
}

func TestDropsFor_MutatingReturnedSlice_DoesNotCorruptRegistry(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"kind":"slime","items":[{"id":"slime-oil","name":"Slime Oil"}]}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	drops := r.DropsFor("slime")
	drops[0].ID = "corrupted"
	fresh := r.DropsFor("slime")
	if fresh[0].ID != "slime-oil" {
		t.Fatalf("registry was mutated via caller's slice: %+v", fresh)
	}
}

func TestAll_ReturnsEveryRegisteredTable(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[
		{"kind":"worm","items":[{"id":"worm-sinew","name":"Worm Sinew"}]},
		{"kind":"leech","items":[{"id":"leech-blood","name":"Leech Blood"}]}
	]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	all := r.All()
	if len(all) != 2 {
		t.Fatalf("expected 2 tables, got %d", len(all))
	}
}
