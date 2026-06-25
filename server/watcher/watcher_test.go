package watcher

import (
	"testing"
	"time"
)

// deterministicRand is a test double for RandSource.
type deterministicRand struct{ val float64 }

func (d *deterministicRand) Float64() float64 { return d.val }

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

// ── NewState ──────────────────────────────────────────────────────────────────

func TestNewStateInitialAlertness(t *testing.T) {
	s := NewState("district-residential")
	if s.Alertness != 20.0 {
		t.Errorf("expected Alertness=20, got %.1f", s.Alertness)
	}
}

func TestNewStateTrustZero(t *testing.T) {
	s := NewState("district-commercial")
	if s.Trust != 0 {
		t.Errorf("expected Trust=0, got %.1f", s.Trust)
	}
}

func TestNewStateBiasNeutral(t *testing.T) {
	s := NewState("district-industrial")
	if s.Bias != BiasNeutral {
		t.Errorf("expected BiasNeutral, got %v", s.Bias)
	}
}

// ── AddAlertness ──────────────────────────────────────────────────────────────

func TestAddAlertnessRaises(t *testing.T) {
	s := NewState("d1")
	s.AddAlertness(20, t0, "test")
	if s.Alertness != 40 {
		t.Errorf("expected 40, got %.1f", s.Alertness)
	}
}

func TestAddAlertnessClampMax(t *testing.T) {
	s := NewState("d1")
	s.AddAlertness(200, t0, "test")
	if s.Alertness != AlertnessMax {
		t.Errorf("expected clamped to %.0f, got %.1f", AlertnessMax, s.Alertness)
	}
}

func TestAddAlertnessEmitsEvent(t *testing.T) {
	s := NewState("d1")
	evts := s.AddAlertness(10, t0, "fo_claim")
	if len(evts) == 0 {
		t.Error("expected at least one event")
	}
	if evts[0].Verb != "ALERT_SPIKE" {
		t.Errorf("expected ALERT_SPIKE, got %q", evts[0].Verb)
	}
}

func TestAddAlertnessSaturationEvent(t *testing.T) {
	s := NewState("d1")
	s.Alertness = 90
	evts := s.AddAlertness(10, t0, "heavy_activity")
	var found bool
	for _, e := range evts {
		if e.Verb == "SATURATION" {
			found = true
		}
	}
	if !found {
		t.Error("expected SATURATION event when crossing 95")
	}
}

// ── ShiftTrust ────────────────────────────────────────────────────────────────

func TestShiftTrustPositive(t *testing.T) {
	s := NewState("d1")
	s.ShiftTrust(30, t0, "community benefit")
	if s.Trust != 30 {
		t.Errorf("expected Trust=30, got %.1f", s.Trust)
	}
}

func TestShiftTrustNegative(t *testing.T) {
	s := NewState("d1")
	s.ShiftTrust(-50, t0, "property damage")
	if s.Trust != -50 {
		t.Errorf("expected Trust=-50, got %.1f", s.Trust)
	}
}

func TestShiftTrustClamped(t *testing.T) {
	s := NewState("d1")
	s.ShiftTrust(-200, t0, "extreme negative")
	if s.Trust != TrustMin {
		t.Errorf("expected min %.0f, got %.1f", TrustMin, s.Trust)
	}
}

// ── Enforcement trigger ───────────────────────────────────────────────────────

func TestEnforcementTrigger(t *testing.T) {
	s := NewState("d1")
	s.Alertness = 70
	s.Trust = -30
	evts := s.checkEnforcementTrigger(t0)
	if len(evts) == 0 {
		t.Error("expected ENFORCEMENT_TRIGGER event")
	}
	if evts[0].Verb != "ENFORCEMENT_TRIGGER" {
		t.Errorf("expected ENFORCEMENT_TRIGGER, got %q", evts[0].Verb)
	}
	if !s.EnforcementTriggered {
		t.Error("EnforcementTriggered should be true")
	}
}

func TestEnforcementNotTriggeredWhenAlertLow(t *testing.T) {
	s := NewState("d1")
	s.Alertness = 40
	s.Trust = -50
	evts := s.checkEnforcementTrigger(t0)
	if len(evts) != 0 {
		t.Error("should not trigger when alertness below threshold")
	}
}

func TestEnforcementNotTriggeredWhenTrustHigh(t *testing.T) {
	s := NewState("d1")
	s.Alertness = 80
	s.Trust = 10
	evts := s.checkEnforcementTrigger(t0)
	if len(evts) != 0 {
		t.Error("should not trigger when trust above -20")
	}
}

func TestIsEnforcementHot(t *testing.T) {
	s := NewState("d1")
	s.Alertness = 70
	s.Trust = -25
	if !s.IsEnforcementHot() {
		t.Error("should be enforcement hot")
	}
}

// ── DecayAlertness ────────────────────────────────────────────────────────────

func TestDecayAlertnessReduces(t *testing.T) {
	s := NewState("d1")
	s.Alertness = 60
	s.DecayAlertness(60*time.Second, t0)
	if s.Alertness >= 60 {
		t.Error("alertness should decrease after 60s decay")
	}
}

func TestDecayAlertnessClampMin(t *testing.T) {
	s := NewState("d1")
	s.Alertness = 0
	s.DecayAlertness(60*time.Second, t0)
	if s.Alertness != 0 {
		t.Errorf("alertness should not go below 0, got %.2f", s.Alertness)
	}
}

// ── ShiftBias ─────────────────────────────────────────────────────────────────

func TestShiftBiasToFrequency(t *testing.T) {
	s := NewState("d1")
	s.ShiftBias(BiasFrequency, 30, t0, "cast_terminal")
	if s.Bias != BiasFrequency {
		t.Errorf("expected Frequency bias, got %v", s.Bias)
	}
}

func TestBiasName(t *testing.T) {
	if BiasName(BiasBloc) != "Bloc" {
		t.Errorf("expected Bloc, got %q", BiasName(BiasBloc))
	}
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestRegistryGetOrCreate(t *testing.T) {
	r := NewRegistry()
	s1 := r.GetOrCreate("district-residential")
	s2 := r.GetOrCreate("district-residential")
	if s1 != s2 {
		t.Error("GetOrCreate should return same instance")
	}
}

func TestRegistryHotDistricts(t *testing.T) {
	r := NewRegistry()
	s := r.GetOrCreate("district-commercial")
	s.Alertness = 70
	s.Trust = -30
	s.checkEnforcementTrigger(t0)
	hot := r.HotDistricts()
	if len(hot) != 1 {
		t.Errorf("expected 1 hot district, got %d", len(hot))
	}
}

func TestRegistryAlertnessByDistrict(t *testing.T) {
	r := NewRegistry()
	r.GetOrCreate("district-residential").Alertness = 45
	r.GetOrCreate("district-commercial").Alertness = 75
	m := r.AlertnessByDistrict()
	if m["district-residential"] != 45 {
		t.Errorf("expected 45, got %.1f", m["district-residential"])
	}
}

// ── Vigilante anomaly system ──────────────────────────────────────────────────

func TestDisruptionDebtAccumulatesAboveHighViz(t *testing.T) {
	s := NewState("d1")
	s.Alertness = AlertHighViz + 20 // 20 points above threshold
	s.AccumulateDisruptionDebt(60*time.Second, t0)
	if s.DisruptionDebt <= 0 {
		t.Error("disruption debt should accumulate when alertness > AlertHighViz")
	}
}

func TestDisruptionDebtDoesNotAccumulateBelowThreshold(t *testing.T) {
	s := NewState("d1")
	s.Alertness = AlertHighViz - 1 // just below threshold
	s.AccumulateDisruptionDebt(60*time.Second, t0)
	if s.DisruptionDebt != 0 {
		t.Errorf("expected no debt below AlertHighViz, got %.2f", s.DisruptionDebt)
	}
}

func TestDisruptionDebtDecaysWhenAlertLow(t *testing.T) {
	s := NewState("d1")
	s.DisruptionDebt = 20.0
	s.Alertness = 30 // well below AlertHighViz
	s.AccumulateDisruptionDebt(60*time.Second, t0)
	if s.DisruptionDebt >= 20.0 {
		t.Error("disruption debt should decay when alertness drops")
	}
}

func TestDisruptionDebtClampedToMax(t *testing.T) {
	s := NewState("d1")
	s.Alertness = AlertnessMax
	s.AccumulateDisruptionDebt(10000*time.Second, t0) // extreme duration
	if s.DisruptionDebt > SpawnDebtMax {
		t.Errorf("disruption debt should not exceed SpawnDebtMax, got %.2f", s.DisruptionDebt)
	}
}

func TestSpawnProbabilityZeroBelowThreshold(t *testing.T) {
	p := spawnProbability(SpawnDebtThreshold - 1)
	if p != 0 {
		t.Errorf("expected 0 probability below threshold, got %.3f", p)
	}
}

func TestSpawnProbabilityPositiveAtThreshold(t *testing.T) {
	p := spawnProbability(SpawnDebtThreshold)
	if p <= 0 {
		t.Errorf("expected positive probability at threshold, got %.3f", p)
	}
}

func TestSpawnProbabilityCappedAtMax(t *testing.T) {
	p := spawnProbability(SpawnDebtMax)
	if p > SpawnProbMax {
		t.Errorf("probability should not exceed SpawnProbMax (%.2f), got %.3f", SpawnProbMax, p)
	}
}

func TestCheckVigilanteSpawnReturnsNilWhenDebtLow(t *testing.T) {
	s := NewState("d1")
	s.DisruptionDebt = SpawnDebtThreshold - 1
	rng := &deterministicRand{val: 0.0} // always rolls lowest
	v, _ := s.CheckVigilanteSpawn(rng, t0)
	if v != nil {
		t.Error("should not spawn when debt below threshold")
	}
}

func TestCheckVigilanteSpawnSucceedsWithSufficientDebt(t *testing.T) {
	s := NewState("d1")
	s.DisruptionDebt = SpawnDebtThreshold + 10
	rng := &deterministicRand{val: 0.0} // always rolls 0.0 — guarantees spawn
	v, evts := s.CheckVigilanteSpawn(rng, t0)
	if v == nil {
		t.Fatal("expected a vigilante to spawn")
	}
	if v.Alignment != "chaotic_neutral" {
		t.Errorf("vigilante should be chaotic_neutral, got %q", v.Alignment)
	}
	if len(evts) == 0 {
		t.Error("expected VIGILANTE_SPAWN event")
	}
	if evts[0].Verb != "VIGILANTE_SPAWN" {
		t.Errorf("expected VIGILANTE_SPAWN verb, got %q", evts[0].Verb)
	}
}

func TestCheckVigilanteSpawnResetsDebt(t *testing.T) {
	s := NewState("d1")
	s.DisruptionDebt = 50.0
	rng := &deterministicRand{val: 0.0} // forces spawn
	s.CheckVigilanteSpawn(rng, t0)
	if s.DisruptionDebt != 0 {
		t.Errorf("debt should reset to 0 after spawn, got %.2f", s.DisruptionDebt)
	}
}

func TestCheckVigilanteSpawnNoSpawnOnHighRoll(t *testing.T) {
	s := NewState("d1")
	s.DisruptionDebt = SpawnDebtThreshold + 5 // probability ~15-25%
	rng := &deterministicRand{val: 0.99}      // rolls near max — above spawn probability
	v, _ := s.CheckVigilanteSpawn(rng, t0)
	if v != nil {
		t.Error("high roll should not produce a spawn")
	}
}

func TestAnomalyTierRequiresHighDebt(t *testing.T) {
	tier := selectTier(AnomalyDebtThreshold - 1)
	if tier == TierAnomaly {
		t.Error("anomaly tier should require debt ≥ AnomalyDebtThreshold")
	}
	tier = selectTier(AnomalyDebtThreshold)
	if tier != TierAnomaly {
		t.Errorf("expected TierAnomaly at AnomalyDebtThreshold, got %d", tier)
	}
}

func TestRiotBreakerSpawnsAtSaturation(t *testing.T) {
	s := NewState("d1")
	s.Alertness = AlertSaturation + 1
	rng := &deterministicRand{val: 0.5}
	archetype := selectArchetype(s, rng)
	if archetype != ArchetypeRiotBreaker {
		t.Errorf("expected RiotBreaker at saturation, got %d", archetype)
	}
}

func TestVigilanteTargetPriorityTrustAdjustment(t *testing.T) {
	// High trust should reduce priority.
	highTrust := VigilanteTargetPriority(50, 20, 100)
	lowTrust := VigilanteTargetPriority(50, 20, -100)
	if highTrust >= lowTrust {
		t.Errorf("high trust (%.2f) should reduce priority vs low trust (%.2f)", highTrust, lowTrust)
	}
}

func TestVigilanteHasValidStats(t *testing.T) {
	s := NewState("d1")
	s.DisruptionDebt = 50.0
	rng := &deterministicRand{val: 0.0} // forces spawn
	v, _ := s.CheckVigilanteSpawn(rng, t0)
	if v == nil {
		t.Fatal("expected spawn")
	}
	if v.HP <= 0 {
		t.Errorf("vigilante HP should be positive, got %d", v.HP)
	}
	if v.Damage <= 0 {
		t.Errorf("vigilante Damage should be positive, got %d", v.Damage)
	}
	if v.Name == "" {
		t.Error("vigilante should have a name")
	}
	if v.Special == "" {
		t.Error("vigilante should have a special ability")
	}
}

func TestTickAccumulatesDebt(t *testing.T) {
	s := NewState("d1")
	s.Alertness = AlertHighViz + 10
	s.Tick(60*time.Second, t0)
	if s.DisruptionDebt <= 0 {
		t.Error("Tick should accumulate disruption debt when alertness is high")
	}
}
