package idunaauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func generateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

// makeToken builds a minimal ES256 JWT signed with the given key.
func makeToken(t *testing.T, priv *ecdsa.PrivateKey, claims map[string]interface{}) string {
	t.Helper()
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256","typ":"JWT"}`))
	payload, _ := json.Marshal(claims)
	payloadEnc := base64.RawURLEncoding.EncodeToString(payload)
	signingInput := header + "." + payloadEnc
	digest := sha256.Sum256([]byte(signingInput))
	r, s, err := ecdsa.Sign(rand.Reader, priv, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	rb := pad32(r.Bytes())
	sb := pad32(s.Bytes())
	sig := append(rb, sb...)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// pad32 pads or trims b to exactly 32 bytes big-endian.
func pad32(b []byte) []byte {
	out := make([]byte, 32)
	if len(b) >= 32 {
		copy(out, b[len(b)-32:])
	} else {
		copy(out[32-len(b):], b)
	}
	return out
}

func futureExp() int64 { return time.Now().Add(time.Hour).Unix() }
func pastExp() int64   { return time.Now().Add(-time.Hour).Unix() }

// ── Verifier.Verify ───────────────────────────────────────────────────────────

func TestVerify_ValidToken(t *testing.T) {
	priv := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&priv.PublicKey})
	tok := makeToken(t, priv, map[string]interface{}{
		"sub": "user-123", "iss": "iduna", "exp": futureExp(),
	})
	c, err := v.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if c.Subject != "user-123" {
		t.Errorf("Subject: got %q, want %q", c.Subject, "user-123")
	}
	if c.Issuer != "iduna" {
		t.Errorf("Issuer: got %q, want %q", c.Issuer, "iduna")
	}
}

func TestVerify_WrongKey(t *testing.T) {
	priv := generateKey(t)
	wrong := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&wrong.PublicKey})
	tok := makeToken(t, priv, map[string]interface{}{"sub": "x", "exp": futureExp()})
	_, err := v.Verify(tok)
	if !errors.Is(err, ErrSignatureInvalid) {
		t.Errorf("wrong key: got %v, want ErrSignatureInvalid", err)
	}
}

func TestVerify_ExpiredToken(t *testing.T) {
	priv := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&priv.PublicKey})
	tok := makeToken(t, priv, map[string]interface{}{"sub": "x", "exp": pastExp()})
	_, err := v.Verify(tok)
	if !errors.Is(err, ErrTokenExpired) {
		t.Errorf("expired: got %v, want ErrTokenExpired", err)
	}
}

func TestVerify_MalformedToken_TwoParts(t *testing.T) {
	priv := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&priv.PublicKey})
	_, err := v.Verify("header.payload")
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("two-part: got %v, want ErrTokenInvalid", err)
	}
}

func TestVerify_MalformedToken_BadBase64Payload(t *testing.T) {
	priv := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&priv.PublicKey})
	_, err := v.Verify("header.!!!badbase64!!!.sig")
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("bad base64: got %v, want ErrTokenInvalid", err)
	}
}

func TestVerify_TamperedPayload(t *testing.T) {
	priv := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&priv.PublicKey})
	tok := makeToken(t, priv, map[string]interface{}{"sub": "alice", "exp": futureExp()})
	// Replace payload with a tampered one
	parts := strings.SplitN(tok, ".", 3)
	tampered := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"bob","exp":` + fmt.Sprint(futureExp()) + `}`))
	tok2 := parts[0] + "." + tampered + "." + parts[2]
	_, err := v.Verify(tok2)
	if !errors.Is(err, ErrSignatureInvalid) {
		t.Errorf("tampered payload: got %v, want ErrSignatureInvalid", err)
	}
}

func TestVerify_NoKeys(t *testing.T) {
	v := &Verifier{client: nil}
	// keys empty, no IDUNA running — will try to refresh and fail OR return ErrNoKeysLoaded
	priv := generateKey(t)
	tok := makeToken(t, priv, map[string]interface{}{"sub": "x", "exp": futureExp()})
	// ensureKeys will try HTTP — we override by calling Verify on v that has no keys and no client
	v2 := WithKeys(nil)
	_, err := v2.Verify(tok)
	if err == nil {
		t.Error("no keys: expected error, got nil")
	}
	_ = v
}

func TestVerify_MultipleKeys_CorrectKeyUsed(t *testing.T) {
	// Load two keys; sign with the second one; verify should succeed.
	k1 := generateKey(t)
	k2 := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&k1.PublicKey, &k2.PublicKey})
	tok := makeToken(t, k2, map[string]interface{}{"sub": "carol", "exp": futureExp()})
	c, err := v.Verify(tok)
	if err != nil {
		t.Fatalf("multi-key: %v", err)
	}
	if c.Subject != "carol" {
		t.Errorf("Subject: got %q, want carol", c.Subject)
	}
}

func TestVerify_ShortSig(t *testing.T) {
	priv := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&priv.PublicKey})
	// Build a token with a 10-byte sig (too short)
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"ES256"}`))
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"x","exp":` + fmt.Sprint(futureExp()) + `}`))
	sig := base64.RawURLEncoding.EncodeToString([]byte("tooshort"))
	_, err := v.Verify(header + "." + payload + "." + sig)
	if !errors.Is(err, ErrTokenInvalid) {
		t.Errorf("short sig: got %v, want ErrTokenInvalid", err)
	}
}

func TestVerify_RawClaims(t *testing.T) {
	priv := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&priv.PublicKey})
	tok := makeToken(t, priv, map[string]interface{}{
		"sub": "dave", "exp": futureExp(), "player_id": "pid-999",
	})
	c, err := v.Verify(tok)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if c.Raw["player_id"] != "pid-999" {
		t.Errorf("Raw claim: got %v, want pid-999", c.Raw["player_id"])
	}
}

// ── WithKeys / NewVerifier ─────────────────────────────────────────────────────

func TestWithKeys_PreloadsKeys(t *testing.T) {
	priv := generateKey(t)
	v := WithKeys([]*ecdsa.PublicKey{&priv.PublicKey})
	v.mu.RLock()
	n := len(v.keys)
	v.mu.RUnlock()
	if n != 1 {
		t.Errorf("WithKeys: got %d keys, want 1", n)
	}
}

func TestNewVerifier_DefaultBaseURL(t *testing.T) {
	v := NewVerifier()
	if !strings.HasPrefix(v.baseURL, "http") {
		t.Errorf("NewVerifier baseURL: got %q", v.baseURL)
	}
}

// ── pad32 edge cases ──────────────────────────────────────────────────────────

func TestPad32_Short(t *testing.T) {
	b := pad32([]byte{1, 2, 3})
	if len(b) != 32 {
		t.Errorf("pad32 short: len=%d, want 32", len(b))
	}
	if b[31] != 3 {
		t.Errorf("pad32 short: last byte=%d, want 3", b[31])
	}
}

func TestPad32_Long(t *testing.T) {
	in := make([]byte, 33)
	in[0] = 0xFF
	in[1] = 0xAA
	for i := 2; i < 33; i++ {
		in[i] = byte(i)
	}
	b := pad32(in)
	if len(b) != 32 {
		t.Errorf("pad32 long: len=%d, want 32", len(b))
	}
}

func TestPad32_Exact32(t *testing.T) {
	in := make([]byte, 32)
	for i := range in {
		in[i] = byte(i + 1)
	}
	b := pad32(in)
	if b[0] != 1 || b[31] != 32 {
		t.Errorf("pad32 exact32: wrong values")
	}
}

// ── big.Int round-trip (simulates JWKS decode) ────────────────────────────────

func TestBigIntRoundTrip(t *testing.T) {
	priv := generateKey(t)
	xBytes := pad32(priv.PublicKey.X.Bytes())
	yBytes := pad32(priv.PublicKey.Y.Bytes())
	xEnc := base64.RawURLEncoding.EncodeToString(xBytes)
	yEnc := base64.RawURLEncoding.EncodeToString(yBytes)
	xDec, _ := base64.RawURLEncoding.DecodeString(xEnc)
	yDec, _ := base64.RawURLEncoding.DecodeString(yEnc)
	x2 := new(big.Int).SetBytes(xDec)
	y2 := new(big.Int).SetBytes(yDec)
	if x2.Cmp(priv.PublicKey.X) != 0 || y2.Cmp(priv.PublicKey.Y) != 0 {
		t.Error("big.Int round-trip through base64url failed")
	}
}
