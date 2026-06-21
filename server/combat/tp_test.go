package combat

import (
	"testing"
)

// --- TPPerHit (pure function) ---

func TestTPPerHit_1HSword_NoHaste(t *testing.T) {
	// 240 du / 6 = 40 TP per hit
	got := TPPerHit(Delay1HSword, 0)
	if got != 40 {
		t.Errorf("1H sword 0%% haste: got %d, want 40", got)
	}
}

func TestTPPerHit_GreatSword_NoHaste(t *testing.T) {
	// 480 du / 6 = 80 TP per hit
	got := TPPerHit(DelayGreatSword, 0)
	if got != 80 {
		t.Errorf("great sword 0%% haste: got %d, want 80", got)
	}
}

func TestTPPerHit_HtH_NoHaste(t *testing.T) {
	// 80 du / 6 = 13 TP per hit
	got := TPPerHit(DelayHtH, 0)
	if got != 13 {
		t.Errorf("HtH 0%% haste: got %d, want 13", got)
	}
}

func TestTPPerHit_HasteReducesGain(t *testing.T) {
	no := TPPerHit(Delay1HSword, 0)
	with := TPPerHit(Delay1HSword, 25)
	if with >= no {
		t.Errorf("25%% haste should reduce TP per hit: no-haste=%d, with-haste=%d", no, with)
	}
}

func TestTPPerHit_HasteCapAt80(t *testing.T) {
	at80 := TPPerHit(Delay1HSword, 80)
	at99 := TPPerHit(Delay1HSword, 99)
	if at80 != at99 {
		t.Errorf("haste above 80%% should be capped: at80=%d, at99=%d", at80, at99)
	}
}

func TestTPPerHit_NegativeHasteClamped(t *testing.T) {
	// negative haste should behave the same as 0%
	neg := TPPerHit(Delay1HSword, -10)
	zero := TPPerHit(Delay1HSword, 0)
	if neg != zero {
		t.Errorf("negative haste should clamp to 0: got %d vs %d", neg, zero)
	}
}

func TestTPPerHit_MinimumFloor(t *testing.T) {
	// Even at 80% haste with a tiny delay (10 du), should be >= MinTPPerHit
	tp := TPPerHit(10, 80)
	if tp < MinTPPerHit {
		t.Errorf("TP per hit should never fall below %d, got %d", MinTPPerHit, tp)
	}
}

func TestTPPerHit_StaffAndPolearm(t *testing.T) {
	staff := TPPerHit(DelayStaff, 0)
	pole := TPPerHit(DelayPolearm, 0)
	if staff >= pole {
		t.Errorf("polearm (longer delay) should give more TP than staff: staff=%d pole=%d", staff, pole)
	}
}

// --- AddTP (state-mutating) ---

func TestAddTP_AccumulatesCorrectly(t *testing.T) {
	s := TPState{}
	tp := s.AddTP(Delay1HSword, 0) // +40
	if tp != 40 || s.Current != 40 {
		t.Errorf("after 1 hit: got %d, want 40", tp)
	}
	tp = s.AddTP(Delay1HSword, 0) // +40 = 80
	if tp != 80 {
		t.Errorf("after 2 hits: got %d, want 80", tp)
	}
}

func TestAddTP_CapsAt300(t *testing.T) {
	s := TPState{Current: 290}
	tp := s.AddTP(Delay1HSword, 0) // +40 would be 330, should cap at 300
	if tp != TPCap {
		t.Errorf("cap: got %d, want %d", tp, TPCap)
	}
	if s.Current != TPCap {
		t.Errorf("s.Current after cap: got %d", s.Current)
	}
}

func TestAddTP_CapsExactlyAt300(t *testing.T) {
	s := TPState{Current: 260}
	s.AddTP(Delay1HSword, 0) // +40 = exactly 300
	if s.Current != 300 {
		t.Errorf("exact cap: got %d", s.Current)
	}
}

func TestAddTP_HasteAffectsGain(t *testing.T) {
	noHaste := TPState{}
	withHaste := TPState{}
	noHaste.AddTP(Delay1HSword, 0)
	withHaste.AddTP(Delay1HSword, 50)
	if withHaste.Current >= noHaste.Current {
		t.Errorf("50%% haste should produce less TP: no=%d with=%d",
			noHaste.Current, withHaste.Current)
	}
}

func TestAddTP_ReturnsNewTotal(t *testing.T) {
	s := TPState{Current: 50}
	returned := s.AddTP(Delay1HSword, 0) // +40 = 90
	if returned != s.Current {
		t.Errorf("return value should equal s.Current: returned=%d s.Current=%d", returned, s.Current)
	}
}

func TestAddTP_GreatSwordFullTP_In2Hits(t *testing.T) {
	// Great sword: 80 TP per hit → need 2 hits for WS threshold (80+80=160 ≥ 100)
	s := TPState{}
	s.AddTP(DelayGreatSword, 0) // 80
	s.AddTP(DelayGreatSword, 0) // 160
	if !s.CanWeaponSkill() {
		t.Errorf("should be able to WS after 2 great sword hits (TP=%d)", s.Current)
	}
}

// --- CanWeaponSkill ---

func TestCanWeaponSkill_FalseBelow100(t *testing.T) {
	s := TPState{Current: 99}
	if s.CanWeaponSkill() {
		t.Error("99 TP: should not be able to WS")
	}
}

func TestCanWeaponSkill_TrueAt100(t *testing.T) {
	s := TPState{Current: 100}
	if !s.CanWeaponSkill() {
		t.Error("100 TP: should be able to WS")
	}
}

func TestCanWeaponSkill_TrueAt300(t *testing.T) {
	s := TPState{Current: 300}
	if !s.CanWeaponSkill() {
		t.Error("300 TP: should be able to WS")
	}
}

// --- UseWeaponSkill ---

func TestUseWeaponSkill_ResetsTPToZero(t *testing.T) {
	s := TPState{Current: 250}
	ok := s.UseWeaponSkill()
	if !ok {
		t.Fatal("UseWeaponSkill at 250 TP should succeed")
	}
	if s.Current != 0 {
		t.Errorf("TP after WS: got %d, want 0", s.Current)
	}
}

func TestUseWeaponSkill_FalseAndNoChangeBelow100(t *testing.T) {
	s := TPState{Current: 80}
	before := s.Current
	ok := s.UseWeaponSkill()
	if ok {
		t.Error("UseWeaponSkill at 80 TP should return false")
	}
	if s.Current != before {
		t.Errorf("TP should not change on failed WS: before=%d after=%d", before, s.Current)
	}
}

func TestUseWeaponSkill_ExactlyAt100(t *testing.T) {
	s := TPState{Current: 100}
	if !s.UseWeaponSkill() {
		t.Error("UseWeaponSkill at exactly 100 TP should succeed")
	}
	if s.Current != 0 {
		t.Errorf("TP after WS at 100: got %d", s.Current)
	}
}

func TestUseWeaponSkill_ZeroTP_Fails(t *testing.T) {
	s := TPState{Current: 0}
	if s.UseWeaponSkill() {
		t.Error("UseWeaponSkill at 0 TP should fail")
	}
}

// --- ConsumeTP ---

func TestConsumeTP_SubtractsCorrectly(t *testing.T) {
	s := TPState{Current: 200}
	ok := s.ConsumeTP(60)
	if !ok {
		t.Fatal("ConsumeTP(60) on 200 TP should succeed")
	}
	if s.Current != 140 {
		t.Errorf("after consuming 60: got %d, want 140", s.Current)
	}
}

func TestConsumeTP_ExactAmount(t *testing.T) {
	s := TPState{Current: 50}
	ok := s.ConsumeTP(50)
	if !ok {
		t.Fatal("ConsumeTP exact should succeed")
	}
	if s.Current != 0 {
		t.Errorf("after consuming all: got %d, want 0", s.Current)
	}
}

func TestConsumeTP_InsufficientReturnsFalse(t *testing.T) {
	s := TPState{Current: 40}
	ok := s.ConsumeTP(50)
	if ok {
		t.Error("ConsumeTP(50) on 40 TP should fail")
	}
	if s.Current != 40 {
		t.Errorf("TP should not change on failed consume: got %d", s.Current)
	}
}

func TestConsumeTP_NegativeAmountFails(t *testing.T) {
	s := TPState{Current: 100}
	ok := s.ConsumeTP(-10)
	if ok {
		t.Error("ConsumeTP with negative amount should fail")
	}
}

func TestConsumeTP_ZeroAmount(t *testing.T) {
	s := TPState{Current: 100}
	ok := s.ConsumeTP(0)
	if !ok {
		t.Error("ConsumeTP(0) should succeed (no-op)")
	}
	if s.Current != 100 {
		t.Errorf("ConsumeTP(0) should not change TP: got %d", s.Current)
	}
}

// --- Reset ---

func TestReset_ClearsTP(t *testing.T) {
	s := TPState{Current: 250}
	s.Reset()
	if s.Current != 0 {
		t.Errorf("after Reset: got %d, want 0", s.Current)
	}
}

func TestReset_OnZeroIsNoOp(t *testing.T) {
	s := TPState{Current: 0}
	s.Reset()
	if s.Current != 0 {
		t.Error("Reset on zero TP should leave it at zero")
	}
}

// --- HitsToWS ---

func TestHitsToWS_1HSword(t *testing.T) {
	// 240 du / 6 = 40 TP per hit → ceil(100/40) = 3 hits
	h := HitsToWS(Delay1HSword, 0)
	if h != 3 {
		t.Errorf("1H sword hits to WS: got %d, want 3", h)
	}
}

func TestHitsToWS_GreatSword(t *testing.T) {
	// 480 du / 6 = 80 TP per hit → ceil(100/80) = 2 hits
	h := HitsToWS(DelayGreatSword, 0)
	if h != 2 {
		t.Errorf("great sword hits to WS: got %d, want 2", h)
	}
}

func TestHitsToWS_HasteIncreasesHitsNeeded(t *testing.T) {
	no := HitsToWS(Delay1HSword, 0)
	with := HitsToWS(Delay1HSword, 50)
	if with <= no {
		t.Errorf("haste should increase hits needed (less TP per hit): no=%d with=%d", no, with)
	}
}

func TestHitsToWS_NeverZero(t *testing.T) {
	h := HitsToWS(1, 0) // tiny delay → MinTPPerHit (5) → ceil(100/5) = 20
	if h <= 0 {
		t.Errorf("HitsToWS should never return 0: got %d", h)
	}
}

// --- Integration: earn TP → WS → earn again ---

func TestFullCycle_EarnAndUseWeaponSkill(t *testing.T) {
	s := TPState{}

	// 1H sword, no haste: 40 TP per hit. 3 hits → 120 TP.
	for i := 0; i < 3; i++ {
		s.AddTP(Delay1HSword, 0)
	}
	if s.Current != 120 {
		t.Fatalf("after 3 hits: got %d, want 120", s.Current)
	}
	if !s.CanWeaponSkill() {
		t.Fatal("should be able to WS at 120 TP")
	}

	// Execute WS.
	if !s.UseWeaponSkill() {
		t.Fatal("UseWeaponSkill failed at 120 TP")
	}
	if s.Current != 0 {
		t.Errorf("TP after WS: got %d, want 0", s.Current)
	}

	// Can't WS again immediately.
	if s.CanWeaponSkill() {
		t.Error("should not be able to WS immediately after using one")
	}

	// Hit again and start accumulating toward next WS.
	s.AddTP(Delay1HSword, 0)
	if s.Current != 40 {
		t.Errorf("first hit after WS: got %d, want 40", s.Current)
	}
}

func TestFullCycle_StoreAtCap(t *testing.T) {
	s := TPState{}
	// Add enough hits to cap at 300.
	for i := 0; i < 10; i++ {
		s.AddTP(Delay1HSword, 0)
	}
	if s.Current != TPCap {
		t.Errorf("after many hits: got %d, want %d", s.Current, TPCap)
	}
	// WS resets to 0 regardless of cap.
	s.UseWeaponSkill()
	if s.Current != 0 {
		t.Errorf("after WS from cap: got %d, want 0", s.Current)
	}
}
