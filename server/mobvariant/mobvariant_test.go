package mobvariant

import "testing"

func TestLoadJSON_RegistersVariant(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"base_kind":"rabbit","display_name":"Fierce Rabbit","power_mul":1.8}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	all := r.All()
	if len(all) != 1 || all[0].DisplayName != "Fierce Rabbit" {
		t.Fatalf("unexpected variants: %+v", all)
	}
}

func TestLoadJSON_ZeroPowerMul_DefaultsToOne(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"base_kind":"rabbit","display_name":"Plain Rabbit","power_mul":0}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	all := r.All()
	if all[0].PowerMul != 1.0 {
		t.Fatalf("expected a zero power_mul to default to 1.0, got %v", all[0].PowerMul)
	}
}

func TestForBaseKind_CaseInsensitiveMatch(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[
		{"base_kind":"Rabbit","display_name":"Fierce Rabbit","power_mul":1.8},
		{"base_kind":"worm","display_name":"Elder Worm","power_mul":1.6}
	]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	matches := r.ForBaseKind("rabbit")
	if len(matches) != 1 || matches[0].DisplayName != "Fierce Rabbit" {
		t.Fatalf("expected 1 case-insensitive match for rabbit, got %+v", matches)
	}
}

func TestForBaseKind_NoMatches_ReturnsEmpty(t *testing.T) {
	r := NewRegistry()
	if err := r.LoadJSON([]byte(`[{"base_kind":"rabbit","display_name":"Fierce Rabbit","power_mul":1.8}]`)); err != nil {
		t.Fatalf("LoadJSON: %v", err)
	}
	if matches := r.ForBaseKind("skeleton"); len(matches) != 0 {
		t.Fatalf("expected no matches for an unregistered base kind, got %+v", matches)
	}
}
