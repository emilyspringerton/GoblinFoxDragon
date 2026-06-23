package mob

import "testing"

// --- NewLeech ---

func TestNewLeech_Kind(t *testing.T) {
	l := NewLeech("l1", Pos{})
	if l.Kind != KindLeech {
		t.Errorf("Kind=%q, want %q", l.Kind, KindLeech)
	}
}

func TestNewLeech_SceneID(t *testing.T) {
	l := NewLeech("l1", Pos{})
	if l.SceneID != 3 {
		t.Errorf("SceneID=%d, want 3 (Swampville)", l.SceneID)
	}
}

func TestNewLeech_Stats(t *testing.T) {
	l := NewLeech("l1", Pos{10, 1, 5})
	if l.HP != LeechHP || l.MaxHP != LeechHP {
		t.Errorf("HP: got %d/%d, want %d", l.HP, l.MaxHP, LeechHP)
	}
	if l.MeleeDamage != LeechDamage {
		t.Errorf("Damage=%d, want %d", l.MeleeDamage, LeechDamage)
	}
	if l.MoveSpeed != LeechMoveSpeed {
		t.Errorf("MoveSpeed=%f", l.MoveSpeed)
	}
}

func TestNewLeech_FastSwing(t *testing.T) {
	// Leeches swing faster than slimes and lizards (nibbler flavour).
	if LeechSwingDelay >= SlimeSwingDelay {
		t.Errorf("leech should swing faster than slime: leech=%v slime=%v", LeechSwingDelay, SlimeSwingDelay)
	}
	if LeechSwingDelay >= LizardSwingDelay {
		t.Errorf("leech should swing faster than lizard: leech=%v lizard=%v", LeechSwingDelay, LizardSwingDelay)
	}
}

func TestNewLeech_AggressiveAggroRange(t *testing.T) {
	l := NewLeech("l1", Pos{})
	if l.AggroRange < 10 {
		t.Errorf("leech should be aggressive: AggroRange=%f", l.AggroRange)
	}
}

// --- NewSlime ---

func TestNewSlime_Kind(t *testing.T) {
	s := NewSlime("s1", Pos{})
	if s.Kind != KindSlime {
		t.Errorf("Kind=%q, want %q", s.Kind, KindSlime)
	}
}

func TestNewSlime_SceneID(t *testing.T) {
	s := NewSlime("s1", Pos{})
	if s.SceneID != 3 {
		t.Errorf("SceneID=%d, want 3", s.SceneID)
	}
}

func TestNewSlime_HighHP(t *testing.T) {
	s := NewSlime("s1", Pos{})
	if s.HP != SlimeHP {
		t.Errorf("HP=%d, want %d", s.HP, SlimeHP)
	}
	// Slimes should have more HP than leeches and lizards.
	if SlimeHP <= LeechHP {
		t.Errorf("slime should have more HP than leech: slime=%d leech=%d", SlimeHP, LeechHP)
	}
	if SlimeHP <= LizardHP {
		t.Errorf("slime should have more HP than lizard: slime=%d lizard=%d", SlimeHP, LizardHP)
	}
}

func TestNewSlime_Slow(t *testing.T) {
	if SlimeMoveSpeed >= LeechMoveSpeed {
		t.Errorf("slime should be slower than leech: slime=%f leech=%f", SlimeMoveSpeed, LeechMoveSpeed)
	}
	if SlimeMoveSpeed >= LizardMoveSpeed {
		t.Errorf("slime should be slower than lizard: slime=%f lizard=%f", SlimeMoveSpeed, LizardMoveSpeed)
	}
}

func TestNewSlime_PassiveLowAggro(t *testing.T) {
	s := NewSlime("s1", Pos{})
	// Passive: very short aggro range.
	if s.AggroRange > 10 {
		t.Errorf("slime aggro range should be short (passive): got %f", s.AggroRange)
	}
}

// --- NewLizard ---

func TestNewLizard_Kind(t *testing.T) {
	l := NewLizard("lz1", Pos{})
	if l.Kind != KindLizard {
		t.Errorf("Kind=%q, want %q", l.Kind, KindLizard)
	}
}

func TestNewLizard_SceneID(t *testing.T) {
	l := NewLizard("lz1", Pos{})
	if l.SceneID != 3 {
		t.Errorf("SceneID=%d, want 3", l.SceneID)
	}
}

func TestNewLizard_FastAndAggressive(t *testing.T) {
	l := NewLizard("lz1", Pos{})
	if l.MoveSpeed <= LeechMoveSpeed {
		t.Errorf("lizard should be faster than leech: lizard=%f leech=%f", l.MoveSpeed, LeechMoveSpeed)
	}
	if l.AggroRange <= LeechAggroRange {
		t.Errorf("lizard should have longer aggro range than leech: lizard=%f leech=%f", l.AggroRange, LeechAggroRange)
	}
}

func TestNewLizard_HighDamage(t *testing.T) {
	if LizardDamage <= LeechDamage {
		t.Errorf("lizard should hit harder than leech: lizard=%d leech=%d", LizardDamage, LeechDamage)
	}
	if LizardDamage <= SlimeDamage {
		t.Errorf("lizard should hit harder than slime: lizard=%d slime=%d", LizardDamage, SlimeDamage)
	}
}

// --- SwampvilleSpawns ---

func TestSwampvilleSpawns_Count(t *testing.T) {
	spawns := SwampvilleSpawns()
	if len(spawns) != 14 {
		t.Errorf("SwampvilleSpawns: want 14 (6 leech + 4 slime + 4 lizard), got %d", len(spawns))
	}
}

func TestSwampvilleSpawns_UniqueIDs(t *testing.T) {
	seen := map[string]bool{}
	for _, m := range SwampvilleSpawns() {
		if seen[m.ID] {
			t.Errorf("duplicate mob ID: %q", m.ID)
		}
		seen[m.ID] = true
	}
}

func TestSwampvilleSpawns_AllScene3(t *testing.T) {
	for _, m := range SwampvilleSpawns() {
		if m.SceneID != 3 {
			t.Errorf("mob %q SceneID=%d, want 3", m.ID, m.SceneID)
		}
	}
}

func TestSwampvilleSpawns_KindCounts(t *testing.T) {
	counts := map[string]int{}
	for _, m := range SwampvilleSpawns() {
		counts[m.Kind]++
	}
	if counts[KindLeech] != 6 {
		t.Errorf("leech count: got %d, want 6", counts[KindLeech])
	}
	if counts[KindSlime] != 4 {
		t.Errorf("slime count: got %d, want 4", counts[KindSlime])
	}
	if counts[KindLizard] != 4 {
		t.Errorf("lizard count: got %d, want 4", counts[KindLizard])
	}
}

func TestSwampvilleSpawns_OutsideTownRadius(t *testing.T) {
	const townRadius = 10.0
	for _, m := range SwampvilleSpawns() {
		d := dist(m.HomePos, Pos{0, m.HomePos.Y, 0})
		if d < townRadius {
			t.Errorf("mob %q too close to town: dist=%.1f < %.1f", m.ID, d, townRadius)
		}
	}
}

func TestSwampvilleSpawns_LizardsOutermost(t *testing.T) {
	spawns := SwampvilleSpawns()
	avgLeech, avgLizard := 0.0, 0.0
	leechN, lizardN := 0, 0
	origin := Pos{}
	for _, m := range spawns {
		d := dist(m.HomePos, origin)
		if m.Kind == KindLeech {
			avgLeech += d
			leechN++
		}
		if m.Kind == KindLizard {
			avgLizard += d
			lizardN++
		}
	}
	avgLeech /= float64(leechN)
	avgLizard /= float64(lizardN)
	if avgLizard <= avgLeech {
		t.Errorf("lizards should be further from town than leeches: lizard avg=%.1f leech avg=%.1f", avgLizard, avgLeech)
	}
}
