package skillchain

import (
	"testing"
	"time"
)

const window = DefaultChainWindow

// --- Tier 1: same-element chains ---

func TestTier1SameElement(t *testing.T) {
	pairs := [][2]Resonance{
		{Liquefaction, Liquefaction},
		{Impaction, Impaction},
		{Detonation, Detonation},
		{Scission, Scission},
		{Reverberation, Reverberation},
		{Induration, Induration},
		{Compression, Compression},
		{Transfixion, Transfixion},
	}
	for _, p := range pairs {
		res, ok := Chain([]Resonance{p[0]}, []Resonance{p[1]}, 0, window)
		if !ok {
			t.Errorf("%v+%v: expected tier1 chain, got none", p[0], p[1])
			continue
		}
		if res.Tier != Tier1 {
			t.Errorf("%v+%v: tier = %d, want 1", p[0], p[1], res.Tier)
		}
		if res.Multiplier != 0.20 {
			t.Errorf("%v+%v: multiplier = %v, want 0.20", p[0], p[1], res.Multiplier)
		}
		if res.Resonance != p[0] {
			t.Errorf("%v+%v: resonance = %v, want %v", p[0], p[1], res.Resonance, p[0])
		}
	}
}

// --- Tier 2: cross-element chains ---

func TestFusion(t *testing.T) {
	cases := [][2]Resonance{
		{Transfixion, Liquefaction},
		{Liquefaction, Transfixion},
		{Liquefaction, Impaction},
	}
	for _, c := range cases {
		res, ok := Chain([]Resonance{c[0]}, []Resonance{c[1]}, 0, window)
		if !ok {
			t.Errorf("%v+%v: expected Fusion, got none", c[0], c[1])
			continue
		}
		if res.Resonance != Fusion {
			t.Errorf("%v+%v: resonance = %v, want Fusion", c[0], c[1], res.Resonance)
		}
		if res.Tier != Tier2 {
			t.Errorf("%v+%v: tier = %d, want 2", c[0], c[1], res.Tier)
		}
	}
}

func TestFragmentation(t *testing.T) {
	cases := [][2]Resonance{
		{Impaction, Detonation},
		{Detonation, Impaction},
		{Detonation, Reverberation},
		{Reverberation, Detonation},
	}
	for _, c := range cases {
		res, ok := Chain([]Resonance{c[0]}, []Resonance{c[1]}, 0, window)
		if !ok {
			t.Errorf("%v+%v: expected Fragmentation, got none", c[0], c[1])
			continue
		}
		if res.Resonance != Fragmentation {
			t.Errorf("%v+%v: resonance = %v, want Fragmentation", c[0], c[1], res.Resonance)
		}
	}
}

func TestGravitation(t *testing.T) {
	cases := [][2]Resonance{
		{Scission, Compression},
		{Compression, Scission},
	}
	for _, c := range cases {
		res, ok := Chain([]Resonance{c[0]}, []Resonance{c[1]}, 0, window)
		if !ok {
			t.Errorf("%v+%v: expected Gravitation, got none", c[0], c[1])
			continue
		}
		if res.Resonance != Gravitation {
			t.Errorf("%v+%v: resonance = %v, want Gravitation", c[0], c[1], res.Resonance)
		}
	}
}

func TestDistortion(t *testing.T) {
	cases := [][2]Resonance{
		{Reverberation, Induration},
		{Induration, Reverberation},
		{Scission, Reverberation},
		{Induration, Scission},
	}
	for _, c := range cases {
		res, ok := Chain([]Resonance{c[0]}, []Resonance{c[1]}, 0, window)
		if !ok {
			t.Errorf("%v+%v: expected Distortion, got none", c[0], c[1])
			continue
		}
		if res.Resonance != Distortion {
			t.Errorf("%v+%v: resonance = %v, want Distortion", c[0], c[1], res.Resonance)
		}
	}
}

func TestTier2Multiplier(t *testing.T) {
	res, ok := Chain([]Resonance{Transfixion}, []Resonance{Liquefaction}, 0, window)
	if !ok {
		t.Fatal("expected Fusion chain")
	}
	if res.Multiplier != 0.35 {
		t.Errorf("tier2 multiplier: got %v, want 0.35", res.Multiplier)
	}
}

// --- Tier 3: compound chains ---

func TestLight(t *testing.T) {
	cases := [][2]Resonance{
		{Fusion, Fragmentation},
		{Fragmentation, Fusion},
	}
	for _, c := range cases {
		res, ok := Chain([]Resonance{c[0]}, []Resonance{c[1]}, 0, window)
		if !ok {
			t.Errorf("%v+%v: expected Light, got none", c[0], c[1])
			continue
		}
		if res.Resonance != Light {
			t.Errorf("%v+%v: resonance = %v, want Light", c[0], c[1], res.Resonance)
		}
		if res.Tier != Tier3 {
			t.Errorf("%v+%v: tier = %d, want 3", c[0], c[1], res.Tier)
		}
		if res.Multiplier != 0.50 {
			t.Errorf("%v+%v: multiplier = %v, want 0.50", c[0], c[1], res.Multiplier)
		}
	}
}

func TestDarkness(t *testing.T) {
	cases := [][2]Resonance{
		{Gravitation, Distortion},
		{Distortion, Gravitation},
	}
	for _, c := range cases {
		res, ok := Chain([]Resonance{c[0]}, []Resonance{c[1]}, 0, window)
		if !ok {
			t.Errorf("%v+%v: expected Darkness, got none", c[0], c[1])
			continue
		}
		if res.Resonance != Darkness {
			t.Errorf("%v+%v: resonance = %v, want Darkness", c[0], c[1], res.Resonance)
		}
	}
}

// --- Multi-attribute WS picks highest tier ---

func TestHighestTierWins(t *testing.T) {
	// WS1 carries Fusion; WS2 carries Fragmentation → Light (tier3)
	// Also WS2 could carry Detonation for a tier2, but tier3 should win.
	ws1 := []Resonance{Fusion, Detonation}
	ws2 := []Resonance{Fragmentation, Impaction}
	res, ok := Chain(ws1, ws2, 0, window)
	if !ok {
		t.Fatal("expected a chain")
	}
	if res.Resonance != Light {
		t.Errorf("should prefer Light (tier3) over Fragmentation (tier2): got %v", res.Resonance)
	}
}

// --- Window expiry ---

func TestWindowExpiredReturnsNoChain(t *testing.T) {
	_, ok := Chain([]Resonance{Liquefaction}, []Resonance{Liquefaction}, 9*time.Second, window)
	if ok {
		t.Error("expired window: expected no chain")
	}
}

func TestWindowExactlyAtBoundaryAllowed(t *testing.T) {
	_, ok := Chain([]Resonance{Liquefaction}, []Resonance{Liquefaction}, window, window)
	if !ok {
		t.Error("exactly at window boundary should still form chain")
	}
}

func TestNegativeElapsedReturnsNoChain(t *testing.T) {
	_, ok := Chain([]Resonance{Liquefaction}, []Resonance{Liquefaction}, -1, window)
	if ok {
		t.Error("negative elapsed should return no chain")
	}
}

// --- No valid combination ---

func TestNoChainForUnrelatedResonances(t *testing.T) {
	_, ok := Chain([]Resonance{Liquefaction}, []Resonance{Compression}, 0, window)
	if ok {
		t.Error("Liquefaction+Compression should form no chain")
	}
}

func TestEmptyAttrsReturnsNoChain(t *testing.T) {
	_, ok := Chain(nil, []Resonance{Liquefaction}, 0, window)
	if ok {
		t.Error("empty ws1 attrs should return no chain")
	}
	_, ok = Chain([]Resonance{Liquefaction}, nil, 0, window)
	if ok {
		t.Error("empty ws2 attrs should return no chain")
	}
}

// --- Canonical weapon skills integrate correctly ---

func TestRedLotusBlade_ShiningBlade_Fusion(t *testing.T) {
	// Red Lotus Blade (Liquefaction+Transfixion) → Shining Blade (Transfixion)
	// The Transfixion+Transfixion pair is tier1, but Liquefaction+Transfixion
	// and Transfixion+Transfixion are both in ws1×ws2; highest tier wins.
	// Red Lotus: [Liq, Tra], Shining: [Tra]
	// Pairs: (Liq,Tra)=Fusion(T2), (Tra,Tra)=Transfixion(T1) → Fusion wins
	ws1 := CanonicalWeaponSkills["Red Lotus Blade"]
	ws2 := CanonicalWeaponSkills["Shining Blade"]
	res, ok := Chain(ws1.Attrs, ws2.Attrs, 0, window)
	if !ok {
		t.Fatal("Red Lotus Blade → Shining Blade: expected chain")
	}
	if res.Resonance != Fusion {
		t.Errorf("expected Fusion, got %v", res.Resonance)
	}
}

func TestStarburst_Sunburst_Light(t *testing.T) {
	// Starburst (Fusion) → Sunburst (Fragmentation) = Light
	ws1 := CanonicalWeaponSkills["Starburst"]
	ws2 := CanonicalWeaponSkills["Sunburst"]
	res, ok := Chain(ws1.Attrs, ws2.Attrs, 0, window)
	if !ok {
		t.Fatal("Starburst → Sunburst: expected chain")
	}
	if res.Resonance != Light {
		t.Errorf("expected Light, got %v", res.Resonance)
	}
}

func TestFrostbite_RockCrusher_Distortion(t *testing.T) {
	// Frostbite (Induration+Reverberation) → Rock Crusher (Reverberation)
	// (Ind,Rev)=Distortion(T2), (Rev,Rev)=Reverberation(T1) → Distortion wins
	ws1 := CanonicalWeaponSkills["Frostbite"]
	ws2 := CanonicalWeaponSkills["Rock Crusher"]
	res, ok := Chain(ws1.Attrs, ws2.Attrs, 0, window)
	if !ok {
		t.Fatal("Frostbite → Rock Crusher: expected chain")
	}
	if res.Resonance != Distortion {
		t.Errorf("expected Distortion, got %v", res.Resonance)
	}
}

// --- Resonance.String ---

func TestResonanceString(t *testing.T) {
	cases := []struct {
		r    Resonance
		want string
	}{
		{None, "None"},
		{Liquefaction, "Liquefaction"},
		{Impaction, "Impaction"},
		{Detonation, "Detonation"},
		{Scission, "Scission"},
		{Reverberation, "Reverberation"},
		{Induration, "Induration"},
		{Compression, "Compression"},
		{Transfixion, "Transfixion"},
		{Fusion, "Fusion"},
		{Fragmentation, "Fragmentation"},
		{Gravitation, "Gravitation"},
		{Distortion, "Distortion"},
		{Light, "Light"},
		{Darkness, "Darkness"},
		{Resonance(99), "Unknown"},
	}
	for _, tc := range cases {
		if got := tc.r.String(); got != tc.want {
			t.Errorf("Resonance(%d).String() = %q, want %q", tc.r, got, tc.want)
		}
	}
}

// =============================================================================
// Magic Burst tests
// =============================================================================

const burstWin = DefaultBurstWindow

func TestCanBurst_Tier1Elements(t *testing.T) {
	cases := []struct {
		res  Resonance
		elem Element
	}{
		{Liquefaction, ElemFire},
		{Impaction, ElemLightning},
		{Detonation, ElemWind},
		{Scission, ElemEarth},
		{Reverberation, ElemWater},
		{Induration, ElemIce},
		{Compression, ElemDark},
		{Transfixion, ElemLight},
	}
	for _, tc := range cases {
		if !CanBurst(tc.res, tc.elem, 0, burstWin) {
			t.Errorf("CanBurst(%v, %v): expected true", tc.res, tc.elem)
		}
	}
}

func TestCanBurst_WrongElementReturnsFalse(t *testing.T) {
	// Liquefaction (Fire) should not burst with Ice
	if CanBurst(Liquefaction, ElemIce, 0, burstWin) {
		t.Error("Liquefaction should not burst with Ice")
	}
}

func TestCanBurst_Tier2AcceptsMultipleElements(t *testing.T) {
	if !CanBurst(Fusion, ElemFire, 0, burstWin) {
		t.Error("Fusion should burst with Fire")
	}
	if !CanBurst(Fusion, ElemLight, 0, burstWin) {
		t.Error("Fusion should burst with Light")
	}
	if CanBurst(Fusion, ElemIce, 0, burstWin) {
		t.Error("Fusion should NOT burst with Ice")
	}
}

func TestCanBurst_Tier3LightAcceptsAll4(t *testing.T) {
	for _, elem := range []Element{ElemFire, ElemWind, ElemLightning, ElemLight} {
		if !CanBurst(Light, elem, 0, burstWin) {
			t.Errorf("Light should burst with element %d", elem)
		}
	}
	if CanBurst(Light, ElemIce, 0, burstWin) {
		t.Error("Light should NOT burst with Ice")
	}
}

func TestCanBurst_Tier3DarknessAcceptsAll4(t *testing.T) {
	for _, elem := range []Element{ElemEarth, ElemIce, ElemWater, ElemDark} {
		if !CanBurst(Darkness, elem, 0, burstWin) {
			t.Errorf("Darkness should burst with element %d", elem)
		}
	}
	if CanBurst(Darkness, ElemFire, 0, burstWin) {
		t.Error("Darkness should NOT burst with Fire")
	}
}

func TestBurstWindowExpired(t *testing.T) {
	if CanBurst(Liquefaction, ElemFire, burstWin+time.Second, burstWin) {
		t.Error("burst window expired: should return false")
	}
}

func TestBurstWindowAtBoundary(t *testing.T) {
	if !CanBurst(Liquefaction, ElemFire, burstWin, burstWin) {
		t.Error("burst at exact window boundary should be allowed")
	}
}

func TestBurstMultiplierValue(t *testing.T) {
	m := Burst(Liquefaction, ElemFire, 0, burstWin)
	if m != BurstMultiplier {
		t.Errorf("burst multiplier: got %v, want %v", m, BurstMultiplier)
	}
}

func TestBurstMultiplierZeroOnMiss(t *testing.T) {
	m := Burst(Liquefaction, ElemIce, 0, burstWin)
	if m != 0 {
		t.Errorf("no burst expected: got %v", m)
	}
}

func TestBurstMultiplierZeroOnExpiry(t *testing.T) {
	m := Burst(Liquefaction, ElemFire, burstWin+time.Second, burstWin)
	if m != 0 {
		t.Errorf("expired burst: got %v", m)
	}
}

func TestBurstElementsNoneReturnsNil(t *testing.T) {
	if elems := BurstElements(None); elems != nil {
		t.Errorf("BurstElements(None): got %v, want nil", elems)
	}
}

// --- End-to-end: skillchain then magic burst ---

func TestEndToEnd_Distortion_IceBurst(t *testing.T) {
	// Step 1: ws1 = Frostbite, ws2 = Rock Crusher → Distortion
	ws1 := CanonicalWeaponSkills["Frostbite"]
	ws2 := CanonicalWeaponSkills["Rock Crusher"]
	chainResult, ok := Chain(ws1.Attrs, ws2.Attrs, 2*time.Second, window)
	if !ok || chainResult.Resonance != Distortion {
		t.Fatalf("expected Distortion chain, got ok=%v res=%v", ok, chainResult.Resonance)
	}

	// Step 2: Blizzard (Ice) cast 5 seconds after chain → burst
	elapsed := 5 * time.Second
	m := Burst(chainResult.Resonance, ElemIce, elapsed, burstWin)
	if m != BurstMultiplier {
		t.Errorf("Distortion + Blizzard burst: got multiplier %v, want %v", m, BurstMultiplier)
	}
}

func TestEndToEnd_Light_FireBurst(t *testing.T) {
	ws1 := CanonicalWeaponSkills["Starburst"]   // Fusion
	ws2 := CanonicalWeaponSkills["Sunburst"]    // Fragmentation
	chainResult, ok := Chain(ws1.Attrs, ws2.Attrs, 1*time.Second, window)
	if !ok || chainResult.Resonance != Light {
		t.Fatalf("expected Light chain, got ok=%v res=%v", ok, chainResult.Resonance)
	}

	// Fire bursts Light (it accepts Fire/Wind/Lightning/Light)
	m := Burst(chainResult.Resonance, ElemFire, 3*time.Second, burstWin)
	if m != BurstMultiplier {
		t.Errorf("Light + Fire burst: got %v, want %v", m, BurstMultiplier)
	}

	// Lightning also bursts Light
	m2 := Burst(chainResult.Resonance, ElemLightning, 3*time.Second, burstWin)
	if m2 != BurstMultiplier {
		t.Errorf("Light + Lightning burst: got %v", m2)
	}

	// Dark does NOT burst Light
	m3 := Burst(chainResult.Resonance, ElemDark, 3*time.Second, burstWin)
	if m3 != 0 {
		t.Errorf("Light + Dark should not burst: got %v", m3)
	}
}
