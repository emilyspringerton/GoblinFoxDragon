package job

import "testing"

// ── All jobs defined ──────────────────────────────────────────────────────────

func TestAllJobs_22Jobs(t *testing.T) {
	if len(AllJobs) != 22 {
		t.Errorf("AllJobs: got %d, want 22", len(AllJobs))
	}
}

func TestAllJobs_NoDuplicates(t *testing.T) {
	seen := map[JobID]bool{}
	for _, j := range AllJobs {
		if seen[j] {
			t.Errorf("duplicate job: %s", j)
		}
		seen[j] = true
	}
}

func TestStatsFor_AllJobsDefined(t *testing.T) {
	for _, j := range AllJobs {
		if _, err := StatsFor(j); err != nil {
			t.Errorf("StatsFor(%s): %v", j, err)
		}
	}
}

func TestStatsFor_UnknownJobError(t *testing.T) {
	if _, err := StatsFor("XXX"); err != ErrUnknownJob {
		t.Errorf("unknown job: got %v, want ErrUnknownJob", err)
	}
}

// ── HP scaling ────────────────────────────────────────────────────────────────

func TestHPAtLevel_L1IsBaseHP(t *testing.T) {
	s, _ := StatsFor(WAR)
	hp, err := HPAtLevel(WAR, 1)
	if err != nil {
		t.Fatalf("HPAtLevel: %v", err)
	}
	if hp != s.BaseHP {
		t.Errorf("WAR L1 HP: got %d, want %d (BaseHP)", hp, s.BaseHP)
	}
}

func TestHPAtLevel_Scaling(t *testing.T) {
	s, _ := StatsFor(WAR)
	hp10, _ := HPAtLevel(WAR, 10)
	expected := s.BaseHP + s.HPPerLevel*9 // L10 = BaseHP + 9 * HPPerLevel
	if hp10 != expected {
		t.Errorf("WAR L10 HP: got %d, want %d", hp10, expected)
	}
}

func TestHPAtLevel_TankHigherThanMage(t *testing.T) {
	pldHP, _ := HPAtLevel(PLD, 50)
	blmHP, _ := HPAtLevel(BLM, 50)
	if pldHP <= blmHP {
		t.Errorf("PLD L50 HP (%d) should exceed BLM L50 HP (%d)", pldHP, blmHP)
	}
}

func TestHPAtLevel_UnknownJobError(t *testing.T) {
	if _, err := HPAtLevel("XXX", 1); err != ErrUnknownJob {
		t.Errorf("unknown job: got %v, want ErrUnknownJob", err)
	}
}

// ── MP scaling ────────────────────────────────────────────────────────────────

func TestMPAtLevel_MeleeJobZero(t *testing.T) {
	mp, err := MPAtLevel(WAR, 50)
	if err != nil {
		t.Fatalf("MPAtLevel WAR: %v", err)
	}
	if mp != 0 {
		t.Errorf("WAR MP should be 0: got %d", mp)
	}
}

func TestMPAtLevel_MageJobScales(t *testing.T) {
	s, _ := StatsFor(WHM)
	mp, _ := MPAtLevel(WHM, 10)
	expected := s.BaseMP + s.MPPerLevel*9
	if mp != expected {
		t.Errorf("WHM L10 MP: got %d, want %d", mp, expected)
	}
}

func TestMPAtLevel_MagicJobsHigherThanMelee(t *testing.T) {
	whmMP, _ := MPAtLevel(WHM, 30)
	warMP, _ := MPAtLevel(WAR, 30)
	if whmMP <= warMP {
		t.Errorf("WHM MP (%d) should exceed WAR MP (%d) at L30", whmMP, warMP)
	}
}

// ── Canonical job IDs present ─────────────────────────────────────────────────

func TestCanonicalJobs_Spot(t *testing.T) {
	required := []JobID{WAR, WHM, BLM, RDM, THF, PLD, DRK, SAM, NIN, DRG, SMN, BLU, GEO, RUN}
	for _, j := range required {
		if _, err := StatsFor(j); err != nil {
			t.Errorf("expected canonical job %s: %v", j, err)
		}
	}
}
