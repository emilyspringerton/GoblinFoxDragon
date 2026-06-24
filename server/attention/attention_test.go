package attention

import (
	"math"
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

func newMeter() *Meter { return NewMeter("fo-1") }

// ── Dog activity gain ─────────────────────────────────────────────────────────

func TestAddDogActivityRaisesAttention(t *testing.T) {
	m := newMeter()
	evs := m.AddDogActivity(3, 10.0, t0)
	if m.Value <= 0 {
		t.Error("expected Attention to rise after dog activity")
	}
	_ = evs
}

func TestAddDogActivitySuperlinear(t *testing.T) {
	m1 := newMeter()
	m3 := newMeter()
	m1.AddDogActivity(1, 10.0, t0)
	m3.AddDogActivity(3, 10.0, t0)
	// 3 dogs should add more than 3× 1 dog (superlinear n^1.3)
	if m3.Value <= m1.Value*3 {
		t.Errorf("expected superlinear gain: 3-dog=%.2f should exceed 3×1-dog=%.2f", m3.Value, m1.Value*3)
	}
}

func TestAddDogActivityZeroDogs(t *testing.T) {
	m := newMeter()
	evs := m.AddDogActivity(0, 10.0, t0)
	if m.Value != 0 || len(evs) != 0 {
		t.Error("0 dogs should not change attention")
	}
}

// ── Thresholds ────────────────────────────────────────────────────────────────

func TestAuditThresholdFires(t *testing.T) {
	m := newMeter()
	evs := m.Add(AuditThreshold, t0, "test")
	if len(evs) == 0 {
		t.Fatal("expected threshold event")
	}
	found := false
	for _, e := range evs {
		if e.Verb == "AUDIT_THRESHOLD" && e.Summon == SummonOversightSect {
			found = true
		}
	}
	if !found {
		t.Errorf("expected AUDIT_THRESHOLD event in %v", evs)
	}
	if !m.IsUnderAudit() {
		t.Error("IsUnderAudit should be true at threshold")
	}
}

func TestVendorThresholdFires(t *testing.T) {
	m := newMeter()
	evs := m.Add(VendorThreshold, t0, "test")
	found := false
	for _, e := range evs {
		if e.Verb == "VENDOR_THRESHOLD" && e.Summon == SummonShadowOperator {
			found = true
		}
	}
	if !found {
		t.Errorf("expected VENDOR_THRESHOLD in %v", evs)
	}
	if !m.IsUnderVendorPressure() {
		t.Error("IsUnderVendorPressure should be true")
	}
}

func TestThresholdNotFiredTwice(t *testing.T) {
	m := newMeter()
	m.Add(AuditThreshold, t0, "first")
	evs2 := m.Add(50, t0, "second")
	for _, e := range evs2 {
		if e.Verb == "AUDIT_THRESHOLD" {
			t.Error("AUDIT_THRESHOLD should not fire twice while already active")
		}
	}
}

func TestAuditClearOnDecay(t *testing.T) {
	m := newMeter()
	m.Add(AuditThreshold, t0, "raise")
	if !m.IsUnderAudit() {
		t.Fatal("should be under audit")
	}
	// Force value below threshold via direct assignment, then trigger via Tick.
	m.Value = AuditThreshold - 0.5
	m.auditActive = true
	evs := m.Tick(time.Second, t0) // decay by 1.0 → crosses below threshold
	found := false
	for _, e := range evs {
		if e.Verb == "AUDIT_CLEAR" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected AUDIT_CLEAR on decay below threshold, got %v", evs)
	}
	if m.IsUnderAudit() {
		t.Error("IsUnderAudit should be false after AUDIT_CLEAR")
	}
}

func TestVendorClearOnDecay(t *testing.T) {
	m := newMeter()
	m.Value = VendorThreshold - 0.5
	m.vendorActive = true
	evs := m.Tick(time.Second, t0)
	found := false
	for _, e := range evs {
		if e.Verb == "VENDOR_CLEAR" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected VENDOR_CLEAR on decay, got %v", evs)
	}
}

// ── Decay ─────────────────────────────────────────────────────────────────────

func TestTickDecaysAttention(t *testing.T) {
	m := newMeter()
	m.Value = 100.0
	m.Tick(10*time.Second, t0)
	expected := 100.0 - BaseDecayRate*10
	if math.Abs(m.Value-expected) > 0.001 {
		t.Errorf("expected %.2f after decay, got %.2f", expected, m.Value)
	}
}

func TestTickFloorAtZero(t *testing.T) {
	m := newMeter()
	m.Value = 1.0
	m.Tick(100*time.Second, t0)
	if m.Value != 0 {
		t.Errorf("expected 0 floor, got %.4f", m.Value)
	}
}

func TestAddCapsAtMax(t *testing.T) {
	m := newMeter()
	m.Add(AttentionMax*10, t0, "overflow")
	if m.Value != AttentionMax {
		t.Errorf("expected cap at %.0f, got %.4f", AttentionMax, m.Value)
	}
}

// ── Ecosystem effects ─────────────────────────────────────────────────────────

func TestEcosystemEffectsAtZero(t *testing.T) {
	m := newMeter()
	mig, tax, contest := m.EcosystemEffects()
	if mig != 0 || math.Abs(tax-1.0) > 0.001 || math.Abs(contest-1.0) > 0.001 {
		t.Errorf("at zero: mig=%.2f tax=%.2f contest=%.2f (want 0, 1.0, 1.0)", mig, tax, contest)
	}
}

func TestEcosystemEffectsAtMax(t *testing.T) {
	m := newMeter()
	m.Value = AttentionMax
	mig, tax, contest := m.EcosystemEffects()
	if math.Abs(mig-2.0) > 0.001 {
		t.Errorf("expected mig=2.0 at max, got %.4f", mig)
	}
	if math.Abs(tax-1.5) > 0.001 {
		t.Errorf("expected tax=1.5 at max, got %.4f", tax)
	}
	if math.Abs(contest-4.0) > 0.001 {
		t.Errorf("expected contest=4.0 at max, got %.4f", contest)
	}
}

func TestEcosystemEffectsAtHalf(t *testing.T) {
	m := newMeter()
	m.Value = AttentionMax / 2
	mig, tax, contest := m.EcosystemEffects()
	if math.Abs(mig-1.0) > 0.001 {
		t.Errorf("expected mig=1.0 at half, got %.4f", mig)
	}
	if math.Abs(tax-1.25) > 0.001 {
		t.Errorf("expected tax=1.25 at half, got %.4f", tax)
	}
	if math.Abs(contest-2.5) > 0.001 {
		t.Errorf("expected contest=2.5 at half, got %.4f", contest)
	}
}

// ── Registry ──────────────────────────────────────────────────────────────────

func TestRegistryGetOrCreate(t *testing.T) {
	reg := NewRegistry()
	m1 := reg.GetOrCreate("fo-1")
	m2 := reg.GetOrCreate("fo-1")
	if m1 != m2 {
		t.Error("GetOrCreate should return same meter for same FO")
	}
}

func TestRegistryGetMissing(t *testing.T) {
	reg := NewRegistry()
	if reg.Get("fo-missing") != nil {
		t.Error("Get should return nil for unknown FO")
	}
}

func TestRegistryTickAll(t *testing.T) {
	reg := NewRegistry()
	m := reg.GetOrCreate("fo-1")
	m.Value = AuditThreshold - 0.5
	m.auditActive = true
	evs := reg.TickAll(time.Second, t0)
	found := false
	for _, e := range evs {
		if e.Verb == "AUDIT_CLEAR" {
			found = true
		}
	}
	if !found {
		t.Error("expected AUDIT_CLEAR from TickAll")
	}
}

func TestRegistryTotalAttention(t *testing.T) {
	reg := NewRegistry()
	reg.GetOrCreate("fo-1").Value = 100
	reg.GetOrCreate("fo-2").Value = 200
	if math.Abs(reg.TotalAttention()-300) > 0.001 {
		t.Errorf("expected total 300, got %.2f", reg.TotalAttention())
	}
}
