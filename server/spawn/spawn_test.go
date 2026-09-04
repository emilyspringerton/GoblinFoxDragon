package spawn

import "testing"

func TestEnabled_NoRuleLoaded_DefaultsTrue(t *testing.T) {
	r := NewRegistry()
	if !r.Enabled(1, "rabbit") {
		t.Fatal("expected default-enabled for an unregistered (zone, kind) pair")
	}
}

func TestEnabled_ExplicitFalse_Disables(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"zone_id":0,"kind":"rabbit","enabled":false}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if r.Enabled(0, "rabbit") {
		t.Fatal("expected rabbit to be disabled in zone 0")
	}
	if !r.Enabled(0, "beetle") {
		t.Fatal("beetle should be unaffected by the rabbit rule")
	}
	if !r.Enabled(1, "rabbit") {
		t.Fatal("rabbit should be enabled in a different zone with no rule for it")
	}
}

func TestEnabled_CaseInsensitiveKind(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"zone_id":1,"kind":"Hills-Wolf","enabled":false}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if r.Enabled(1, "hills-wolf") {
		t.Fatal("expected case-insensitive match to disable hills-wolf")
	}
}

func TestEnabled_ExplicitTrue_MatchesDefault(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"zone_id":2,"kind":"skeleton","enabled":true}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if !r.Enabled(2, "skeleton") {
		t.Fatal("expected explicit enabled:true to allow spawning")
	}
}

func TestAll_ReturnsEveryRule(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[
		{"zone_id":0,"kind":"rabbit","enabled":true},
		{"zone_id":1,"kind":"beetle","enabled":false}
	]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if len(r.All()) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(r.All()))
	}
}

// GFD-NM-123: "the mob spawn interface can optionally specify a notorious monster spawn for one
// of the base mobs."

func TestNMFor_NoNMBlock_ReturnsNil(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"zone_id":3,"kind":"slime","enabled":true}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if got := r.NMFor(3, "slime"); got != nil {
		t.Fatalf("expected nil NM for a rule with no nm block, got %+v", got)
	}
}

func TestNMFor_WithNMBlock_ReturnsRealConfig(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"zone_id":3,"kind":"slime","enabled":true,"nm":{
		"id":"nm-slime-king","spawn_chance":0.2,"window_open_sec":300,"window_close_sec":1800,"respawn_minutes":60
	}}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	got := r.NMFor(3, "slime")
	if got == nil {
		t.Fatal("expected a real NM config, got nil")
	}
	if got.ID != "nm-slime-king" || got.SpawnChance != 0.2 || got.WindowOpenSec != 300 ||
		got.WindowCloseSec != 1800 || got.RespawnMinutes != 60 {
		t.Fatalf("unexpected NM config: %+v", got)
	}
}

func TestNMFor_CaseInsensitiveKind(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"zone_id":1,"kind":"Rabbit","enabled":true,"nm":{"id":"nm-x","spawn_chance":1}}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if got := r.NMFor(1, "rabbit"); got == nil || got.ID != "nm-x" {
		t.Fatalf("expected case-insensitive NM lookup to find nm-x, got %+v", got)
	}
}

func TestNMFor_DifferentZoneSameKind_Unaffected(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"zone_id":3,"kind":"slime","enabled":true,"nm":{"id":"nm-slime-king","spawn_chance":0.2}}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if got := r.NMFor(0, "slime"); got != nil {
		t.Fatalf("expected no NM for slime in a different zone, got %+v", got)
	}
}

func TestAll_CarriesNMBlockThrough(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[
		{"zone_id":3,"kind":"slime","enabled":true,"nm":{"id":"nm-slime-king","spawn_chance":0.2}},
		{"zone_id":1,"kind":"beetle","enabled":true}
	]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	var sawNM, sawPlain bool
	for _, rule := range r.All() {
		if rule.Kind == "slime" {
			if rule.NM == nil || rule.NM.ID != "nm-slime-king" {
				t.Fatalf("expected slime's rule to carry its NM block, got %+v", rule)
			}
			sawNM = true
		}
		if rule.Kind == "beetle" {
			if rule.NM != nil {
				t.Fatalf("expected beetle's rule to have no NM block, got %+v", rule.NM)
			}
			sawPlain = true
		}
	}
	if !sawNM || !sawPlain {
		t.Fatalf("expected to see both rules, sawNM=%v sawPlain=%v", sawNM, sawPlain)
	}
}
