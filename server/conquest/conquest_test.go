package conquest

import "testing"

// ── NationName ────────────────────────────────────────────────────────────────

func TestNationName_Known(t *testing.T) {
	cases := map[Nation]string{
		NationNeutral:  "Neutral",
		NationSandoria: "San d'Oria",
		NationBastok:   "Bastok",
		NationWindurst: "Windurst",
	}
	for n, want := range cases {
		if got := NationName(n); got != want {
			t.Errorf("NationName(%d)=%q, want %q", n, got, want)
		}
	}
}

// ── Region.AddPoints ─────────────────────────────────────────────────────────

func TestAddPoints_Accumulates(t *testing.T) {
	r := NewRegion(0, "Ronfaure")
	_ = r.AddPoints(NationSandoria, 50)
	_ = r.AddPoints(NationSandoria, 30)
	if r.Points[NationSandoria] != 80 {
		t.Errorf("points=%d, want 80", r.Points[NationSandoria])
	}
}

func TestAddPoints_ClampedAtZero(t *testing.T) {
	r := NewRegion(0, "x")
	_ = r.AddPoints(NationBastok, -100)
	if r.Points[NationBastok] != 0 {
		t.Errorf("negative points should clamp to 0: got %d", r.Points[NationBastok])
	}
}

func TestAddPoints_UnknownNationReturnsError(t *testing.T) {
	r := NewRegion(0, "x")
	if err := r.AddPoints(Nation(99), 10); err != ErrUnknownNation {
		t.Errorf("unknown nation: got %v, want ErrUnknownNation", err)
	}
}

// ── Region.Tick ───────────────────────────────────────────────────────────────

func TestTick_MajorityWins(t *testing.T) {
	r := NewRegion(0, "x")
	_ = r.AddPoints(NationSandoria, 100)
	_ = r.AddPoints(NationBastok, 50)
	ctrl := r.Tick()
	if ctrl != NationSandoria {
		t.Errorf("majority winner: got %v, want Sandoria", ctrl)
	}
}

func TestTick_TieGoesToIncumbent(t *testing.T) {
	r := NewRegion(0, "x")
	r.Controller = NationBastok
	_ = r.AddPoints(NationSandoria, 100)
	_ = r.AddPoints(NationBastok, 100)
	ctrl := r.Tick()
	if ctrl != NationBastok {
		t.Errorf("tie: incumbent Bastok should retain control, got %v", ctrl)
	}
}

func TestTick_NeutralIncumbentOnZeroPoints(t *testing.T) {
	r := NewRegion(0, "x") // Controller = Neutral, zero points
	ctrl := r.Tick()
	if ctrl != NationNeutral {
		t.Errorf("zero points: should stay neutral, got %v", ctrl)
	}
}

func TestTick_ResetsPointsAfterEvaluation(t *testing.T) {
	r := NewRegion(0, "x")
	_ = r.AddPoints(NationSandoria, 200)
	r.Tick()
	for _, n := range AllNations() {
		if r.Points[n] != 0 {
			t.Errorf("points not reset after Tick: nation=%d pts=%d", n, r.Points[n])
		}
	}
}

func TestTick_ControllerUpdates(t *testing.T) {
	r := NewRegion(0, "x")
	r.Controller = NationSandoria
	_ = r.AddPoints(NationBastok, 500)
	ctrl := r.Tick()
	if ctrl != NationBastok {
		t.Errorf("controller should change to Bastok: got %v", ctrl)
	}
	if r.Controller != NationBastok {
		t.Errorf("r.Controller not updated: got %v", r.Controller)
	}
}

func TestTick_SingleNationEarnsControl(t *testing.T) {
	r := NewRegion(0, "x") // starts neutral
	_ = r.AddPoints(NationWindurst, 1)
	ctrl := r.Tick()
	if ctrl != NationWindurst {
		t.Errorf("single earner: got %v, want Windurst", ctrl)
	}
}

// ── Map ───────────────────────────────────────────────────────────────────────

func TestMap_AddAndGet(t *testing.T) {
	m := NewMap()
	m.AddRegion(NewRegion(0, "Ronfaure"))
	r, ok := m.Get(0)
	if !ok || r.Name != "Ronfaure" {
		t.Error("Get after AddRegion failed")
	}
}

func TestMap_GetUnknownReturnsFalse(t *testing.T) {
	m := NewMap()
	_, ok := m.Get(99)
	if ok {
		t.Error("Get unknown should return false")
	}
}

func TestMap_AddPointsUnknownRegion(t *testing.T) {
	m := NewMap()
	if err := m.AddPoints(99, NationBastok, 10); err != ErrUnknownRegion {
		t.Errorf("unknown region: got %v, want ErrUnknownRegion", err)
	}
}

func TestMap_TickAll(t *testing.T) {
	m := NewMap()
	m.AddRegion(NewRegion(0, "A"))
	m.AddRegion(NewRegion(1, "B"))
	_ = m.AddPoints(0, NationSandoria, 100)
	_ = m.AddPoints(1, NationBastok, 200)
	results := m.TickAll()
	if results[0] != NationSandoria {
		t.Errorf("region 0: got %v, want Sandoria", results[0])
	}
	if results[1] != NationBastok {
		t.Errorf("region 1: got %v, want Bastok", results[1])
	}
}

func TestMap_RegionCount(t *testing.T) {
	m := NewMap()
	r0 := NewRegion(0, "A")
	r0.Controller = NationSandoria
	r1 := NewRegion(1, "B")
	r1.Controller = NationSandoria
	r2 := NewRegion(2, "C")
	r2.Controller = NationBastok
	m.AddRegion(r0)
	m.AddRegion(r1)
	m.AddRegion(r2)
	counts := m.RegionCount()
	if counts[NationSandoria] != 2 {
		t.Errorf("Sandoria regions: got %d, want 2", counts[NationSandoria])
	}
	if counts[NationBastok] != 1 {
		t.Errorf("Bastok regions: got %d, want 1", counts[NationBastok])
	}
}

func TestMap_Scoreboard(t *testing.T) {
	m := NewMap()
	m.AddRegion(NewRegion(0, "A"))
	m.AddRegion(NewRegion(1, "B"))
	_ = m.AddPoints(0, NationSandoria, 100)
	_ = m.AddPoints(1, NationSandoria, 50)
	_ = m.AddPoints(1, NationBastok, 30)
	sb := m.Scoreboard()
	if sb[NationSandoria] != 150 {
		t.Errorf("Sandoria total: got %d, want 150", sb[NationSandoria])
	}
	if sb[NationBastok] != 30 {
		t.Errorf("Bastok total: got %d, want 30", sb[NationBastok])
	}
}

func TestDefaultRegions_FourRegions(t *testing.T) {
	if len(DefaultRegions()) != 4 {
		t.Errorf("DefaultRegions: want 4, got %d", len(DefaultRegions()))
	}
}

func TestDefaultRegions_AllNeutral(t *testing.T) {
	for _, r := range DefaultRegions() {
		if r.Controller != NationNeutral {
			t.Errorf("region %s should start neutral: got %v", r.Name, r.Controller)
		}
	}
}
