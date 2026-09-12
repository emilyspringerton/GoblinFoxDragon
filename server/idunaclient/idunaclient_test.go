package idunaclient

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// newTestClient points a real Client at a local httptest.Server -- this package's only
// constructor (New()) reads IDUNA_BASE_URL from the environment, so that's how every test here
// redirects it, same technique already used in apps2/server-go's own
// TestFetchCharacterCombatStatsFallsBackOnUnreachableIDUNA.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	oldURL := os.Getenv("IDUNA_BASE_URL")
	t.Cleanup(func() { os.Setenv("IDUNA_BASE_URL", oldURL) })
	os.Setenv("IDUNA_BASE_URL", srv.URL)
	return New()
}

// CreditGold is new (2026-07-31, EMILY/BACKLOG.md "unify the backends" follow-up) -- the
// symmetric counterpart DeductGold never had. This is the package's first test file at all
// (DeductGold and every other existing method were shipped with zero coverage); backfilling
// those is a separate, larger pass, not attempted here -- scoped to the new method only.

func TestCreditGoldSuccess(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.CreditGold("char-1", 250); err != nil {
		t.Fatalf("CreditGold: unexpected error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", gotMethod)
	}
	if gotPath != "/api/v1/characters/char-1/gold/credit" {
		t.Errorf("expected /api/v1/characters/char-1/gold/credit, got %s", gotPath)
	}
	if gotBody != `{"credit":250}` {
		t.Errorf(`expected body {"credit":250}, got %s`, gotBody)
	}
}

func TestCreditGoldNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.CreditGold("no-such-char", 100); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// TestDoPerformsRealLoginExchangeNotRawSecret (2026-07-31, REDGARDEN_GUI_NORTHSTAR.md Milestone
// 3): confirms the real bug fix -- do() used to send agentSecret itself as the Bearer token,
// which IDUNA's real jwt.Verify-based RequireAuth middleware has always rejected with 401
// (confirmed live against the running IDUNA service, not just theorized). It must now POST
// /api/v1/auth/agent first and send the resulting access_token instead.
func TestDoPerformsRealLoginExchangeNotRawSecret(t *testing.T) {
	var loginCalled bool
	var gotAuthHeaderOnRealCall string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/agent" {
			loginCalled = true
			var body struct {
				AgentName   string `json:"agent_name"`
				AgentSecret string `json:"agent_secret"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.AgentName != "DRAGONSNSHIT-MUD" || body.AgentSecret != "shh-its-a-secret" {
				t.Errorf("login body = %+v, want real agent_name/agent_secret", body)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access_token": "a-real-signed-jwt-not-the-raw-secret",
				"expires_in":   3600,
			})
			return
		}
		gotAuthHeaderOnRealCall = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	oldURL, oldName, oldSecret := os.Getenv("IDUNA_BASE_URL"), os.Getenv("IDUNA_AGENT_NAME"), os.Getenv("IDUNA_AGENT_SECRET")
	t.Cleanup(func() {
		os.Setenv("IDUNA_BASE_URL", oldURL)
		os.Setenv("IDUNA_AGENT_NAME", oldName)
		os.Setenv("IDUNA_AGENT_SECRET", oldSecret)
	})
	os.Setenv("IDUNA_BASE_URL", srv.URL)
	os.Setenv("IDUNA_AGENT_NAME", "DRAGONSNSHIT-MUD")
	os.Setenv("IDUNA_AGENT_SECRET", "shh-its-a-secret")
	c := New()

	_, _ = c.GetCharacter("some-char-id") // 404 from the mock server, irrelevant -- we're checking the header it sent

	if !loginCalled {
		t.Fatal("expected a real POST /api/v1/auth/agent login call before the real request")
	}
	if gotAuthHeaderOnRealCall != "Bearer a-real-signed-jwt-not-the-raw-secret" {
		t.Errorf("Authorization header on the real call = %q, want the JWT from login, not the raw agent secret", gotAuthHeaderOnRealCall)
	}
}

// TestDoCachesTokenAcrossCalls confirms ensureToken doesn't re-login on every single request --
// the JWT is cached and reused until it's within jwtRefreshMargin of its real expiry.
func TestDoCachesTokenAcrossCalls(t *testing.T) {
	loginCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/auth/agent" {
			loginCount++
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "cached-jwt", "expires_in": 3600})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	oldURL, oldName, oldSecret := os.Getenv("IDUNA_BASE_URL"), os.Getenv("IDUNA_AGENT_NAME"), os.Getenv("IDUNA_AGENT_SECRET")
	t.Cleanup(func() {
		os.Setenv("IDUNA_BASE_URL", oldURL)
		os.Setenv("IDUNA_AGENT_NAME", oldName)
		os.Setenv("IDUNA_AGENT_SECRET", oldSecret)
	})
	os.Setenv("IDUNA_BASE_URL", srv.URL)
	os.Setenv("IDUNA_AGENT_NAME", "DRAGONSNSHIT-MUD")
	os.Setenv("IDUNA_AGENT_SECRET", "shh-its-a-secret")
	c := New()

	_, _ = c.GetCharacter("char-1")
	_, _ = c.GetCharacter("char-2")
	_, _ = c.GetCharacter("char-3")

	if loginCount != 1 {
		t.Errorf("login called %d times across 3 requests, want exactly 1 (cached)", loginCount)
	}
}

func TestCreditGoldServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest) // e.g. over the per-call cap
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	err := c.CreditGold("char-1", 99999)
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected an error wrapping ErrServer for a 400 response, got %v", err)
	}
}

// MintBattlegroundsTicket is new (2026-07-31, REDGARDEN_GUI_NORTHSTAR.md Milestone 3).

func TestMintBattlegroundsTicketSuccess(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ticket":     "deadbeef",
			"expires_at": 1234567890,
			"player_id":  "player-1",
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	ticket, err := c.MintBattlegroundsTicket("player-1")
	if err != nil {
		t.Fatalf("MintBattlegroundsTicket: unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/api/v1/redgarden/player-ticket" {
		t.Errorf("expected /api/v1/redgarden/player-ticket, got %s", gotPath)
	}
	if gotBody != `{"player_id":"player-1"}` {
		t.Errorf(`expected body {"player_id":"player-1"}, got %s`, gotBody)
	}
	if ticket.Ticket != "deadbeef" || ticket.ExpiresAt != 1234567890 || ticket.PlayerID != "player-1" {
		t.Errorf("ticket = %+v, unexpected fields", ticket)
	}
}

func TestMintBattlegroundsTicketNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.MintBattlegroundsTicket("no-such-player"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// Hats -- kanban WTHS-012010 ("there is already a hat shop in town that could be a proxy to the
// BrawlPit hat shop allowing you to purchase brawlpit hats with GFD flow from the GFD town").

func TestListHatsSuccess(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]Hat{
			{HatID: "hat-1", Name: "Joystick Cap", Description: "test", FlowCost: 150, ImageAsset: "🎮"},
			{HatID: "hat-2", Name: "Top Hat", Description: "test", FlowCost: 250, ImageAsset: "🎩"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	hats, err := c.ListHats()
	if err != nil {
		t.Fatalf("ListHats: unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/api/v1/hats" {
		t.Errorf("expected /api/v1/hats, got %s", gotPath)
	}
	if len(hats) != 2 || hats[0].HatID != "hat-1" || hats[1].FlowCost != 250 {
		t.Errorf("unexpected hats: %+v", hats)
	}
}

func TestBuyHatSuccess(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.BuyHat("char-1", "hat-1"); err != nil {
		t.Fatalf("BuyHat: unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/api/v1/characters/char-1/hats/buy" {
		t.Errorf("expected /api/v1/characters/char-1/hats/buy, got %s", gotPath)
	}
	if gotBody != `{"hat_id":"hat-1"}` {
		t.Errorf(`expected body {"hat_id":"hat-1"}, got %s`, gotBody)
	}
}

func TestBuyHatInsufficientFlow(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.BuyHat("char-1", "hat-1"); err != ErrInsufficientGold {
		t.Fatalf("expected ErrInsufficientGold, got %v", err)
	}
}

func TestBuyHatNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.BuyHat("char-1", "no-such-hat"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestListCharacterHatsSuccess(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]OwnedHat{
			{HatID: "hat-1", Name: "Joystick Cap", Equipped: true, AcquiredAt: "2026-09-03T00:00:00Z"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	hats, err := c.ListCharacterHats("char-1")
	if err != nil {
		t.Fatalf("ListCharacterHats: unexpected error: %v", err)
	}
	if gotPath != "/api/v1/characters/char-1/hats" {
		t.Errorf("expected /api/v1/characters/char-1/hats, got %s", gotPath)
	}
	if len(hats) != 1 || !hats[0].Equipped {
		t.Errorf("unexpected hats: %+v", hats)
	}
}

// TestGetInventorySuccess -- S252-00's real read path, and the deliberate route choice: hits
// /materials, not /inventory (that one is IDUNA's own separate slot-based bag system).
func TestGetInventorySuccess(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"materials": map[string]int{"earth-crystal": 3}})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	inv, err := c.GetInventory("char-1")
	if err != nil {
		t.Fatalf("GetInventory: unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/api/v1/characters/char-1/materials" {
		t.Errorf("expected /api/v1/characters/char-1/materials, got %s", gotPath)
	}
	if inv["earth-crystal"] != 3 || len(inv) != 1 {
		t.Errorf("unexpected inventory: %+v", inv)
	}
}

// TestGetInventoryEmptyIsNonNilMap -- a character with nothing stored yet must come back as an
// empty, non-nil map, matching every real call site's own "safe to range over" assumption.
func TestGetInventoryEmptyIsNonNilMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"materials": nil})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	inv, err := c.GetInventory("char-1")
	if err != nil {
		t.Fatalf("GetInventory: unexpected error: %v", err)
	}
	if inv == nil {
		t.Fatal("expected a non-nil empty map, got nil")
	}
	if len(inv) != 0 {
		t.Errorf("expected empty map, got %+v", inv)
	}
}

// TestSetInventorySuccess -- S252-01's real write path: a whole-map upsert, PUT to /materials.
func TestSetInventorySuccess(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.SetInventory("char-1", map[string]int{"earth-crystal": 2}); err != nil {
		t.Fatalf("SetInventory: unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("expected PUT, got %s", gotMethod)
	}
	if gotPath != "/api/v1/characters/char-1/materials" {
		t.Errorf("expected /api/v1/characters/char-1/materials, got %s", gotPath)
	}
	if gotBody != `{"materials":{"earth-crystal":2}}` {
		t.Errorf(`unexpected body: %s`, gotBody)
	}
}

func TestSetInventoryServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.SetInventory("char-1", map[string]int{"x": 1}); !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

// GFD-124433: per-job leveling ("when you are a lvl 10 warrior in gfd and you switch to RDM for
// the first time you go back to lvl 1... separate lvls/job").

func TestGetJobLevelsSuccess(t *testing.T) {
	var gotPath, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{
			{"job": "WAR", "level": 10, "current_xp": 500},
			{"job": "RDM", "level": 1, "current_xp": 0},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	levels, err := c.GetJobLevels("char-1")
	if err != nil {
		t.Fatalf("GetJobLevels: unexpected error: %v", err)
	}
	if gotMethod != http.MethodGet {
		t.Errorf("expected GET, got %s", gotMethod)
	}
	if gotPath != "/api/v1/characters/char-1/job-levels" {
		t.Errorf("expected /api/v1/characters/char-1/job-levels, got %s", gotPath)
	}
	if len(levels) != 2 || levels["WAR"].Level != 10 || levels["RDM"].Level != 1 {
		t.Errorf("unexpected job levels: %+v", levels)
	}
}

// TestGetJobLevelsEmptyIsNonNilMap -- a character with no per-job history yet (brand new, or
// predating this feature) must come back as an empty, non-nil map, matching GetInventory's own
// established convention for exactly this shape of "nothing yet" response.
func TestGetJobLevelsEmptyIsNonNilMap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]any{})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	levels, err := c.GetJobLevels("char-1")
	if err != nil {
		t.Fatalf("GetJobLevels: unexpected error: %v", err)
	}
	if levels == nil {
		t.Fatal("expected a non-nil empty map, got nil")
	}
	if len(levels) != 0 {
		t.Errorf("expected empty map, got %+v", levels)
	}
}

func TestUpdateJobLevelSuccess(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.UpdateJobLevel("char-1", "RDM", 5, 120); err != nil {
		t.Fatalf("UpdateJobLevel: unexpected error: %v", err)
	}
	if gotMethod != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", gotMethod)
	}
	if gotPath != "/api/v1/characters/char-1/job-levels/RDM" {
		t.Errorf("expected /api/v1/characters/char-1/job-levels/RDM, got %s", gotPath)
	}
	if gotBody != `{"current_xp":120,"level":5}` {
		t.Errorf("unexpected body: %s", gotBody)
	}
}

func TestUpdateJobLevelServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.UpdateJobLevel("char-1", "WAR", 10, 0); !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
}

// SSH_TRANSPORT_IDENTITY_SPEC.md §3 / Stage 5 client tests. testFingerprintWithSlash matches the
// real shape ssh.FingerprintSHA256 produces (unpadded standard base64 -- "/" and "+" both legal)
// specifically to catch a regression back to sending it as a path segment instead of a query
// parameter.
const testFingerprintWithSlash = "SHA256:ab/cd+ef01234567890123456789012345678901234"

func TestResolveSSHFingerprintSuccess(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.Query().Get("fingerprint")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"character_id": "char-42"})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	charID, err := c.ResolveSSHFingerprint(testFingerprintWithSlash)
	if err != nil {
		t.Fatalf("ResolveSSHFingerprint: unexpected error: %v", err)
	}
	if gotPath != "/api/v1/ssh-keys" {
		t.Errorf("expected /api/v1/ssh-keys, got %s", gotPath)
	}
	// The real proof this went out as a query param, not a mangled path segment: the fingerprint
	// (with its real "/") round-trips exactly through Go's own URL query decoding.
	if gotQuery != testFingerprintWithSlash {
		t.Errorf("fingerprint query param = %q, want %q", gotQuery, testFingerprintWithSlash)
	}
	if charID != "char-42" {
		t.Errorf("expected char-42, got %q", charID)
	}
}

func TestResolveSSHFingerprintNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.ResolveSSHFingerprint("SHA256:never-bound"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestBindSSHKeySuccess(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.BindSSHKey("char-1", testFingerprintWithSlash, "ssh-ed25519 AAAA..."); err != nil {
		t.Fatalf("BindSSHKey: unexpected error: %v", err)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("expected POST, got %s", gotMethod)
	}
	if gotPath != "/api/v1/characters/char-1/ssh-keys" {
		t.Errorf("expected /api/v1/characters/char-1/ssh-keys, got %s", gotPath)
	}
	if gotBody != `{"fingerprint":"`+testFingerprintWithSlash+`","public_key":"ssh-ed25519 AAAA..."}` {
		t.Errorf("unexpected body: %s", gotBody)
	}
}

func TestBindSSHKeyConflict(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.BindSSHKey("char-1", "SHA256:taken", "ssh-ed25519 AAAA..."); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected ErrConflict (fingerprint bound to a different character), got %v", err)
	}
}

func TestListSSHKeysSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]string{
			{"fingerprint": "SHA256:one", "public_key": "ssh-ed25519 one", "created_at": "2026-09-12T00:00:00Z"},
			{"fingerprint": "SHA256:two", "public_key": "ssh-ed25519 two", "created_at": "2026-09-12T00:00:00Z"},
		})
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	keys, err := c.ListSSHKeys("char-1")
	if err != nil {
		t.Fatalf("ListSSHKeys: unexpected error: %v", err)
	}
	if len(keys) != 2 || keys[0].Fingerprint != "SHA256:one" || keys[1].Fingerprint != "SHA256:two" {
		t.Errorf("unexpected keys: %+v", keys)
	}
}

func TestRevokeSSHKeySuccess(t *testing.T) {
	var gotPath, gotMethod, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotQuery = r.URL.Query().Get("fingerprint")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if err := c.RevokeSSHKey("char-1", testFingerprintWithSlash); err != nil {
		t.Fatalf("RevokeSSHKey: unexpected error: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("expected DELETE, got %s", gotMethod)
	}
	if gotPath != "/api/v1/characters/char-1/ssh-keys" {
		t.Errorf("expected /api/v1/characters/char-1/ssh-keys, got %s", gotPath)
	}
	if gotQuery != testFingerprintWithSlash {
		t.Errorf("fingerprint query param = %q, want %q", gotQuery, testFingerprintWithSlash)
	}
}
