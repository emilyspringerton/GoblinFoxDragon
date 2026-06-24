// Package trapxapi exposes the TRAPX city simulation state over HTTP.
//
// Endpoints:
//
//	GET  /api/v1/trapx/city-state    — snapshot of all city districts
//	POST /api/v1/trapx/events        — Emily Prime Dragon fires a city event
//
// Emily Prime RSI OBSERVE phase reads city-state each cycle to detect
// districts in crisis (Rogue Swarm, enforcement escalation, media spike).
// The Dragon ACT phase fires events to escalate city tension.
//
// Start with:
//
//	srv := trapxapi.New(foReg, attnReg, intReg, techClock, cityLedger)
//	http.ListenAndServe(":7071", srv)
package trapxapi

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"dragonsnshit/server/attention"
	"dragonsnshit/server/fieldoffice"
	"dragonsnshit/server/integrity"
	"dragonsnshit/server/ledger"
	"dragonsnshit/server/techpressure"
)

// DistrictState is the per-district snapshot returned in city-state.
type DistrictState struct {
	DistrictID       string  `json:"district_id"`
	ControlIntegrity float64 `json:"control_integrity"`
	IsRogue          bool    `json:"is_rogue"`
	ScarCount        int     `json:"scar_count"`
	Attention        float64 `json:"attention"`
	IsUnderAudit     bool    `json:"is_under_audit"`
	FOCount          int     `json:"fo_count"`
	TotalFlow        float64 `json:"total_flow"`
}

// CityState is the full snapshot returned by GET /api/v1/trapx/city-state.
type CityState struct {
	At             time.Time       `json:"at"`
	TechPressure   float64         `json:"tech_pressure"`
	TechTier       string          `json:"tech_tier"`
	CrownFired     bool            `json:"crown_fired"`
	RogueCount     int             `json:"rogue_count"`
	Districts      []DistrictState `json:"districts"`
	RecentReceipts []string        `json:"recent_receipts"` // last 10 ledger entries
}

// EventRequest is the body for POST /api/v1/trapx/events.
type EventRequest struct {
	EventType  string `json:"event_type"`  // rogue_swarm, media_spike, enforcement_escalation, pressure_spike, bird_correction
	DistrictID string `json:"district_id"` // target district (empty for city-wide)
	Params     string `json:"params"`      // free-form detail string
	Actor      string `json:"actor"`       // who fired it (e.g. "emily-dragon")
}

// EventResponse is the body returned after POST /api/v1/trapx/events.
type EventResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	AppleID string `json:"apple_id,omitempty"`
}

// Server is the TRAPX HTTP API.
type Server struct {
	foReg      *fieldoffice.Registry
	attnReg    *attention.Registry
	intReg     *integrity.Registry
	techClock  *techpressure.Clock
	cityLedger *ledger.Ledger
}

// New creates a TRAPX API server.
func New(
	foReg *fieldoffice.Registry,
	attnReg *attention.Registry,
	intReg *integrity.Registry,
	techClock *techpressure.Clock,
	cityLedger *ledger.Ledger,
) *Server {
	return &Server{foReg: foReg, attnReg: attnReg, intReg: intReg, techClock: techClock, cityLedger: cityLedger}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v1/trapx/city-state":
		s.handleCityState(w, r)
	case r.Method == http.MethodPost && r.URL.Path == "/api/v1/trapx/events":
		s.handleFireEvent(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCityState(w http.ResponseWriter, r *http.Request) {
	now := time.Now()

	// Build per-district states from foReg + attnReg + intReg.
	districtFOs := make(map[string][]*fieldoffice.FieldOffice)
	for _, fo := range s.foReg.All() {
		did := districtIDForFO(fo)
		districtFOs[did] = append(districtFOs[did], fo)
	}

	allDistricts := []string{
		"district-residential", "district-commercial", "district-industrial",
		"district-underground", "district-abandoned",
	}

	var districts []DistrictState
	rogueCount := 0
	for _, did := range allDistricts {
		st := s.intReg.Get(did)
		ci := 1.0
		isRogue := false
		scarCount := 0
		if st != nil {
			ci = st.CI
			isRogue = st.IsRogue
			scarCount = st.ScarCount()
			if isRogue {
				rogueCount++
			}
		}

		attn := 0.0
		underAudit := false
		if m := s.attnReg.Get(did); m != nil {
			attn = m.Value
			underAudit = m.IsUnderAudit()
		}

		totalFlow := 0.0
		fos := districtFOs[did]
		for _, fo := range fos {
			totalFlow += fo.Flow
		}

		districts = append(districts, DistrictState{
			DistrictID:       did,
			ControlIntegrity: ci,
			IsRogue:          isRogue,
			ScarCount:        scarCount,
			Attention:        attn,
			IsUnderAudit:     underAudit,
			FOCount:          len(fos),
			TotalFlow:        totalFlow,
		})
	}

	// Most distressed districts first.
	sort.Slice(districts, func(i, j int) bool {
		return districts[i].ControlIntegrity < districts[j].ControlIntegrity
	})

	// Last 10 receipts.
	all := s.cityLedger.All()
	var recent []string
	start := len(all) - 10
	if start < 0 {
		start = 0
	}
	for _, rec := range all[start:] {
		recent = append(recent, rec.String())
	}

	tier := techpressure.TierForPressure(s.techClock.Pressure)
	state := CityState{
		At:             now,
		TechPressure:   s.techClock.Pressure,
		TechTier:       techpressure.TierName(tier),
		CrownFired:     s.techClock.CrownFired,
		RogueCount:     rogueCount,
		Districts:      districts,
		RecentReceipts: recent,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func (s *Server) handleFireEvent(w http.ResponseWriter, r *http.Request) {
	var req EventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"ok":false,"message":"invalid JSON"}`, http.StatusBadRequest)
		return
	}
	now := time.Now()

	var msg string
	switch strings.ToLower(req.EventType) {
	case "rogue_swarm":
		did := req.DistrictID
		if did == "" {
			did = "district-residential"
		}
		st := s.intReg.GetOrCreate(did)
		// Force CI below rogue threshold to trigger.
		if !st.IsRogue {
			// Simulate jammer use to drive CI down.
			for i := 0; i < 4; i++ {
				st.RecordJammerUse(now)
			}
		}
		s.cityLedger.Append(ledger.VerbRogueSwarm, did, req.Actor, "", req.Params, now)
		msg = "rogue_swarm triggered in " + did

	case "media_spike":
		did := req.DistrictID
		if did == "" {
			did = "district-commercial"
		}
		m := s.attnReg.GetOrCreate(did)
		m.Add(200.0, now, "dragon:media_spike")
		s.cityLedger.Append(ledger.VerbType("MEDIA_SPIKE"), did, req.Actor, "", req.Params, now)
		msg = "media_spike fired in " + did

	case "enforcement_escalation":
		techEvts := s.techClock.AddTierUnlock(3, now)
		for _, e := range techEvts {
			_ = e // already tracked in clock
		}
		s.cityLedger.Append(ledger.VerbType("ENFORCEMENT_ESCALATION"), req.DistrictID, req.Actor, "", req.Params, now)
		msg = "enforcement_escalation: tech pressure raised by tier unlock spike"

	case "pressure_spike":
		for i := 0; i < 10; i++ {
			s.techClock.AddDogDeploy(now)
		}
		s.cityLedger.Append(ledger.VerbType("PRESSURE_SPIKE"), req.DistrictID, req.Actor, "", req.Params, now)
		msg = "pressure_spike: +10 dog deploy spikes added to tech pressure"

	case "bird_correction":
		did := req.DistrictID
		if did == "" {
			did = "district-residential"
		}
		st := s.intReg.GetOrCreate(did)
		st.BirdCorrection(now)
		s.techClock.BirdCorrection(now)
		s.cityLedger.Append(ledger.VerbBirdCorrect, did, req.Actor, "", req.Params, now)
		msg = "bird_correction applied to " + did + " and tech pressure"

	default:
		resp := EventResponse{OK: false, Message: "unknown event_type: " + req.EventType}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := EventResponse{OK: true, Message: msg}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// districtIDForFO maps a FieldOffice to its district ID based on scene ID.
// Scenes 200–299 are TRAPX city scenes; the district is derived from scene group.
func districtIDForFO(fo *fieldoffice.FieldOffice) string {
	switch fo.SceneID {
	case 200:
		return "district-residential"
	case 201:
		return "district-commercial"
	case 202:
		return "district-industrial"
	case 203:
		return "district-underground"
	case 204:
		return "district-abandoned"
	default:
		return "district-residential"
	}
}
