package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"dragonsnshit/server/idunaclient"
)

// fakeSkillsIduna is a minimal in-memory IDUNA skills endpoint (GET list, PATCH add delta).
type fakeSkillsIduna struct {
	mu     sync.Mutex
	skills map[string]float64
	fail   bool
}

func (f *fakeSkillsIduna) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail {
		http.Error(w, "down", http.StatusInternalServerError)
		return
	}
	switch r.Method {
	case http.MethodPatch:
		var b struct {
			SkillName string  `json:"skill_name"`
			Delta     float64 `json:"delta"`
		}
		_ = json.NewDecoder(r.Body).Decode(&b)
		f.skills[b.SkillName] = math.Min(110, f.skills[b.SkillName]+b.Delta)
		w.WriteHeader(http.StatusNoContent)
	default:
		type row struct {
			SkillName string  `json:"skill_name"`
			Value     float64 `json:"value"`
		}
		var rows []row
		for k, v := range f.skills {
			rows = append(rows, row{k, v})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"skills": rows})
	}
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// Fishing (and mining) skill must reach IDUNA, survive a "logout", and survive IDUNA being down
// for a while (the gain is retried, not dropped).
func TestFishingSkillPersistsAcrossLogoutAndOutage(t *testing.T) {
	fake := &fakeSkillsIduna{skills: map[string]float64{}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	t.Setenv("IDUNA_BASE_URL", srv.URL)
	t.Setenv("IDUNA_AGENT_SECRET", "")

	oldGW := gw
	gw = &world{iduna: idunaclient.New(), charIDBySlot: map[string]string{"s1": "char-1"}}
	t.Cleanup(func() { gw = oldGW })

	p := newTestPlayer()
	p.slot = "s1"

	// Fish: skill goes up locally, the tick sync sends the delta.
	p.fishingSkill += 0.5
	p.fishingSkill += 0.1
	syncGatherSkills(p)
	if !approx(fake.skills["fishing"], 0.6) {
		t.Fatalf("IDUNA fishing skill = %v, want 0.6", fake.skills["fishing"])
	}

	// IDUNA outage: more gains accumulate locally and must NOT be marked synced.
	fake.fail = true
	p.fishingSkill += 0.5
	p.miningSkill += 0.1
	syncGatherSkills(p)
	if !approx(p.syncedFishingSkill, 0.6) {
		t.Fatalf("a failed save must not advance the synced baseline (got %v)", p.syncedFishingSkill)
	}
	fake.fail = false
	syncGatherSkills(p) // next tick: retried
	if !approx(fake.skills["fishing"], 1.1) || !approx(fake.skills["mining"], 0.1) {
		t.Fatalf("after recovery IDUNA has fishing=%v mining=%v, want 1.1 / 0.1", fake.skills["fishing"], fake.skills["mining"])
	}

	// Logout/login: a fresh player loads the stored values the way every connect path does.
	loaded, err := gw.iduna.GetSkills("char-1")
	if err != nil {
		t.Fatal(err)
	}
	p2 := newTestPlayer()
	p2.fishingSkill, p2.miningSkill = loaded["fishing"], loaded["mining"]
	p2.syncedFishingSkill, p2.syncedMiningSkill = p2.fishingSkill, p2.miningSkill
	if !approx(p2.fishingSkill, 1.1) {
		t.Errorf("fishing skill after relog = %v, want 1.1", p2.fishingSkill)
	}
	if fishingSkillDelta(p2) != 0 {
		t.Errorf("freshly loaded skill should have no pending delta")
	}
}
