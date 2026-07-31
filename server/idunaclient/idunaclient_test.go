package idunaclient

import (
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
