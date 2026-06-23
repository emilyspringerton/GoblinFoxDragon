// Package idunaauth validates IDUNA-issued ES256 JWTs for DragonsNShit server-go.
//
// # How it works
//
// 1. On first use (or refresh), the Verifier fetches the JWKS from IDUNA
//    (GET {IDUNA_BASE_URL}/.well-known/jwks.json) and caches the EC public keys.
// 2. Incoming JWT strings (from PACKET_CONNECT payload) are decoded and the
//    ES256 signature is verified against the cached key.
// 3. Claims are returned so the game server can load the character record.
//
// # Configuration
//
// Set IDUNA_BASE_URL in the environment (default: http://localhost:8080).
// Verifier is goroutine-safe once constructed; callers may share one instance.
package idunaauth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// Claims is the decoded JWT payload returned on successful validation.
type Claims struct {
	Subject string // "sub" — IDUNA user ID
	Issuer  string // "iss"
	Exp     int64  // "exp" unix timestamp
	Raw     map[string]interface{} // full decoded claims
}

var (
	ErrTokenExpired     = errors.New("jwt: token expired")
	ErrTokenInvalid     = errors.New("jwt: invalid token")
	ErrNoKeysLoaded     = errors.New("jwt: no public keys loaded from IDUNA JWKS")
	ErrSignatureInvalid = errors.New("jwt: signature verification failed")
)

// jwksKey is one entry from /.well-known/jwks.json.
type jwksKey struct {
	Kty string `json:"kty"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	Kid string `json:"kid"`
}

// jwksResponse is the JSON shape of /.well-known/jwks.json.
type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

// Verifier holds cached IDUNA public keys and verifies JWT tokens.
// Safe for concurrent use once created.
type Verifier struct {
	mu        sync.RWMutex
	keys      []*ecdsa.PublicKey
	lastFetch time.Time
	baseURL   string
	client    *http.Client
}

// NewVerifier returns a Verifier that fetches keys from IDUNA_BASE_URL.
// Keys are fetched lazily on first Verify call.
func NewVerifier() *Verifier {
	base := os.Getenv("IDUNA_BASE_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	return &Verifier{
		baseURL: base,
		client:  &http.Client{Timeout: 5 * time.Second},
	}
}

// RefreshKeys fetches fresh JWKS from IDUNA and caches the EC public keys.
func (v *Verifier) RefreshKeys() error {
	url := v.baseURL + "/.well-known/jwks.json"
	resp, err := v.client.Get(url)
	if err != nil {
		return fmt.Errorf("idunaauth: JWKS fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("idunaauth: JWKS status %d", resp.StatusCode)
	}
	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("idunaauth: JWKS decode: %w", err)
	}

	var keys []*ecdsa.PublicKey
	for _, k := range jwks.Keys {
		if k.Kty != "EC" || k.Crv != "P-256" {
			continue
		}
		xb, e1 := base64.RawURLEncoding.DecodeString(k.X)
		yb, e2 := base64.RawURLEncoding.DecodeString(k.Y)
		if e1 != nil || e2 != nil {
			continue
		}
		keys = append(keys, &ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(xb),
			Y:     new(big.Int).SetBytes(yb),
		})
	}

	v.mu.Lock()
	v.keys = keys
	v.lastFetch = time.Now()
	v.mu.Unlock()
	return nil
}

// ensureKeys loads keys if not yet fetched or if stale (>1 hour).
func (v *Verifier) ensureKeys() error {
	v.mu.RLock()
	stale := len(v.keys) == 0 || time.Since(v.lastFetch) > time.Hour
	v.mu.RUnlock()
	if !stale {
		return nil
	}
	return v.RefreshKeys()
}

// Verify validates a JWT token string and returns the decoded Claims.
// The token must be an ES256 JWT signed by an IDUNA private key.
func (v *Verifier) Verify(tokenStr string) (*Claims, error) {
	if err := v.ensureKeys(); err != nil {
		return nil, err
	}

	parts := strings.Split(strings.TrimSpace(tokenStr), ".")
	if len(parts) != 3 {
		return nil, ErrTokenInvalid
	}

	// Decode payload claims.
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("%w: payload base64: %v", ErrTokenInvalid, err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(payloadBytes, &raw); err != nil {
		return nil, fmt.Errorf("%w: payload JSON: %v", ErrTokenInvalid, err)
	}

	// Check expiry.
	if exp, ok := raw["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, ErrTokenExpired
		}
	}

	// Decode signature (ES256 = R||S, each 32 bytes big-endian).
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sigBytes) != 64 {
		return nil, fmt.Errorf("%w: sig base64", ErrTokenInvalid)
	}
	r := new(big.Int).SetBytes(sigBytes[:32])
	s := new(big.Int).SetBytes(sigBytes[32:])

	// Signing input is the first two dot-separated parts.
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))

	// Try each cached public key.
	v.mu.RLock()
	keys := v.keys
	v.mu.RUnlock()
	if len(keys) == 0 {
		return nil, ErrNoKeysLoaded
	}
	for _, pub := range keys {
		if ecdsa.Verify(pub, digest[:], r, s) {
			c := &Claims{Raw: raw}
			if sub, ok := raw["sub"].(string); ok {
				c.Subject = sub
			}
			if iss, ok := raw["iss"].(string); ok {
				c.Issuer = iss
			}
			if exp, ok := raw["exp"].(float64); ok {
				c.Exp = int64(exp)
			}
			return c, nil
		}
	}
	return nil, ErrSignatureInvalid
}

// WithKeys returns a new Verifier pre-loaded with the given public keys.
// Useful for testing without an IDUNA instance.
func WithKeys(keys []*ecdsa.PublicKey) *Verifier {
	v := &Verifier{client: &http.Client{Timeout: 5 * time.Second}}
	v.keys = keys
	v.lastFetch = time.Now()
	return v
}
