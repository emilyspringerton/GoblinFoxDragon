package mob

import "testing"

func TestApplyVariant_ScalesHPAndDamage(t *testing.T) {
	template := Mob{ID: "rabbit-hills-0", Kind: "rabbit", Pos: Pos{X: 15, Y: 8, Z: 5}, HP: 45, MaxHP: 45, MeleeDamage: 4}
	v := ApplyVariant(template, "rabbit-hills-0-variant-fierce-rabbit", "Fierce Rabbit", 1.8)

	if v.HP != 81 || v.MaxHP != 81 {
		t.Fatalf("expected HP/MaxHP scaled to 81 (45*1.8), got HP=%d MaxHP=%d", v.HP, v.MaxHP)
	}
	if v.MeleeDamage != 7 { // int(4*1.8) = 7
		t.Fatalf("expected MeleeDamage scaled to 7, got %d", v.MeleeDamage)
	}
}

func TestApplyVariant_PreservesRealKindForLootAndQuestLookup(t *testing.T) {
	template := Mob{ID: "worm-meadow-0", Kind: "worm", HP: 90, MeleeDamage: 8}
	v := ApplyVariant(template, "worm-meadow-0-variant-elder-worm", "Elder Worm", 1.6)
	if v.Kind != "worm" {
		t.Fatalf("expected Kind to stay 'worm' (variant inherits base loot table), got %q", v.Kind)
	}
}

func TestApplyVariant_SetsDisplayNameAndLabel(t *testing.T) {
	template := Mob{ID: "worm-meadow-0", Kind: "worm", HP: 90}
	v := ApplyVariant(template, "id", "Elder Worm", 1.5)
	if v.DisplayName != "Elder Worm" {
		t.Fatalf("expected DisplayName set, got %q", v.DisplayName)
	}
	if v.Label() != "Elder Worm" {
		t.Fatalf("expected Label() to return the DisplayName, got %q", v.Label())
	}
}

func TestLabel_FallsBackToKindWhenNoDisplayNameSet(t *testing.T) {
	m := Mob{ID: "worm-meadow-0", Kind: "worm"}
	if m.Label() != "worm" {
		t.Fatalf("expected Label() to fall back to Kind, got %q", m.Label())
	}
}

func TestApplyVariant_OffsetsPositionNearTemplate(t *testing.T) {
	template := Mob{ID: "x", Kind: "rabbit", Pos: Pos{X: 15, Y: 8, Z: 5}, HP: 45}
	v := ApplyVariant(template, "id", "Fierce Rabbit", 1.5)
	if v.Pos.X == template.Pos.X && v.Pos.Z == template.Pos.Z {
		t.Fatal("expected the variant to spawn at a real, offset position, not exactly on top of its template")
	}
	dx, dz := v.Pos.X-template.Pos.X, v.Pos.Z-template.Pos.Z
	if dx*dx+dz*dz > 25 { // sanity: "pretty close to" means a few units, not a different zone
		t.Fatalf("expected the variant to spawn close to its template, got offset (%v, %v)", dx, dz)
	}
}
