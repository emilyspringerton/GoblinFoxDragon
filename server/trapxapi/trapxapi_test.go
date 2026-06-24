package trapxapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dragonsnshit/server/attention"
	"dragonsnshit/server/fieldoffice"
	"dragonsnshit/server/integrity"
	"dragonsnshit/server/ledger"
	"dragonsnshit/server/techpressure"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

func newTestServer() *Server {
	foReg := fieldoffice.NewRegistry()
	for _, fo := range fieldoffice.DefaultFieldOffices(nil) {
		foReg.Add(fo)
	}
	attnReg := attention.NewRegistry()
	intReg := integrity.NewRegistry()
	for _, id := range []string{"district-residential", "district-commercial", "district-industrial", "district-underground", "district-abandoned"} {
		intReg.GetOrCreate(id)
	}
	tc := techpressure.NewClock()
	cl := ledger.NewLedger()
	return New(foReg, attnReg, intReg, tc, cl)
}

// ── GET /api/v1/trapx/city-state ─────────────────────────────────────────────

func TestCityStateReturns200(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trapx/city-state", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestCityStateContentType(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trapx/city-state", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	ct := w.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("expected application/json, got %q", ct)
	}
}

func TestCityStateBodyDecodable(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trapx/city-state", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var cs CityState
	if err := json.NewDecoder(w.Body).Decode(&cs); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
}

func TestCityStateHasDistricts(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trapx/city-state", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var cs CityState
	json.NewDecoder(w.Body).Decode(&cs)
	if len(cs.Districts) != 5 {
		t.Errorf("expected 5 districts, got %d", len(cs.Districts))
	}
}

func TestCityStateTechPressureZero(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trapx/city-state", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var cs CityState
	json.NewDecoder(w.Body).Decode(&cs)
	if cs.TechPressure != 0 {
		t.Errorf("expected zero initial tech pressure, got %.1f", cs.TechPressure)
	}
}

func TestCityStateRogueCountZero(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trapx/city-state", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var cs CityState
	json.NewDecoder(w.Body).Decode(&cs)
	if cs.RogueCount != 0 {
		t.Errorf("expected 0 rogue districts initially, got %d", cs.RogueCount)
	}
}

func TestCityStateCIIsOne(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trapx/city-state", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var cs CityState
	json.NewDecoder(w.Body).Decode(&cs)
	for _, d := range cs.Districts {
		if d.ControlIntegrity != 1.0 {
			t.Errorf("district %s: expected CI=1.0, got %.3f", d.DistrictID, d.ControlIntegrity)
		}
	}
}

func TestCityState404Unknown(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/trapx/unknown", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── POST /api/v1/trapx/events ─────────────────────────────────────────────────

func postEvent(t *testing.T, srv *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trapx/events",
		strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

func TestFireEventRogueSwarm(t *testing.T) {
	srv := newTestServer()
	w := postEvent(t, srv, `{"event_type":"rogue_swarm","district_id":"district-underground","actor":"emily-dragon"}`)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp EventResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.OK {
		t.Errorf("expected ok=true, got %+v", resp)
	}
}

func TestFireEventMediaSpike(t *testing.T) {
	srv := newTestServer()
	w := postEvent(t, srv, `{"event_type":"media_spike","district_id":"district-commercial","actor":"emily-dragon"}`)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	// Verify attention raised in district-commercial.
	if m := srv.attnReg.Get("district-commercial"); m == nil || m.Value <= 0 {
		t.Error("media_spike should raise attention in district-commercial")
	}
}

func TestFireEventEnforcementEscalation(t *testing.T) {
	srv := newTestServer()
	w := postEvent(t, srv, `{"event_type":"enforcement_escalation","actor":"emily-dragon"}`)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if srv.techClock.Pressure <= 0 {
		t.Error("enforcement_escalation should raise tech pressure")
	}
}

func TestFireEventPressureSpike(t *testing.T) {
	srv := newTestServer()
	w := postEvent(t, srv, `{"event_type":"pressure_spike","actor":"emily-dragon"}`)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if srv.techClock.Pressure <= 0 {
		t.Error("pressure_spike should raise tech pressure")
	}
}

func TestFireEventBirdCorrection(t *testing.T) {
	srv := newTestServer()
	// Raise pressure first.
	for i := 0; i < 20; i++ {
		srv.techClock.AddDogDeploy(t0)
	}
	before := srv.techClock.Pressure
	postEvent(t, srv, `{"event_type":"bird_correction","district_id":"district-residential","actor":"emily-dragon"}`)
	if srv.techClock.Pressure >= before {
		t.Errorf("bird_correction should lower pressure: before=%.1f after=%.1f", before, srv.techClock.Pressure)
	}
}

func TestFireEventUnknown(t *testing.T) {
	srv := newTestServer()
	w := postEvent(t, srv, `{"event_type":"unknown_event","actor":"emily-dragon"}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for unknown event_type, got %d", w.Code)
	}
}

func TestFireEventBadJSON(t *testing.T) {
	srv := newTestServer()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/trapx/events",
		strings.NewReader(`{not json`))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", w.Code)
	}
}

func TestRogueSwarmRecordedInLedger(t *testing.T) {
	srv := newTestServer()
	postEvent(t, srv, `{"event_type":"rogue_swarm","district_id":"district-abandoned","actor":"emily-dragon"}`)
	all := srv.cityLedger.All()
	if len(all) == 0 {
		t.Fatal("expected ledger entry after rogue_swarm event")
	}
	last := all[len(all)-1]
	if last.FOID != "district-abandoned" {
		t.Errorf("expected FOID=district-abandoned, got %q", last.FOID)
	}
}
