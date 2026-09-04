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
