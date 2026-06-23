// Package idunaclient is a thin HTTP client for IDUNA MMO API calls from GFD server-go.
//
// Reads IDUNA_BASE_URL (default http://localhost:8080) and authenticates with
// IDUNA_AGENT_SECRET (Bearer token) set by server operator.
// All calls block; callers should run them in goroutines if needed.
package idunaclient

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"
)

var (
	ErrNotFound         = errors.New("idunaclient: not found")
	ErrInsufficientGold = errors.New("idunaclient: insufficient gold balance")
	ErrConflict         = errors.New("idunaclient: conflict")
	ErrServer           = errors.New("idunaclient: server error")
)

// Character is the IDUNA character record returned by GET /api/v1/characters/:id.
type Character struct {
	CharacterID string  `json:"character_id"`
	PlayerID    string  `json:"player_id"`
	Name        string  `json:"name"`
	SceneID     int     `json:"scene_id"`
	PosX        float64 `json:"pos_x"`
	PosY        float64 `json:"pos_y"`
	PosZ        float64 `json:"pos_z"`
	GoldBalance int     `json:"gold_balance"`
	Level       int     `json:"level"`
	CurrentXP   int     `json:"current_xp"`
	JobMain     string  `json:"job_main"`
	JobSub      string  `json:"job_sub"`
}

// Client calls the IDUNA MMO API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New returns a Client reading IDUNA_BASE_URL and IDUNA_AGENT_SECRET from env.
func New() *Client {
	base := os.Getenv("IDUNA_BASE_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	return &Client{
		baseURL: base,
		token:   os.Getenv("IDUNA_AGENT_SECRET"),
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.http.Do(req)
}

// GetCharacter fetches the character record by character_id.
func (c *Client) GetCharacter(characterID string) (*Character, error) {
	req, _ := http.NewRequest(http.MethodGet, c.baseURL+"/api/v1/characters/"+characterID, nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("idunaclient: GetCharacter: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
	var ch Character
	if err := json.NewDecoder(resp.Body).Decode(&ch); err != nil {
		return nil, fmt.Errorf("idunaclient: GetCharacter decode: %w", err)
	}
	return &ch, nil
}

// UpdatePosition patches the character's scene_id and position.
func (c *Client) UpdatePosition(characterID string, sceneID int, x, y, z float64) error {
	body, _ := json.Marshal(map[string]interface{}{
		"scene_id": sceneID, "pos_x": x, "pos_y": y, "pos_z": z,
	})
	req, _ := http.NewRequest(http.MethodPatch,
		c.baseURL+"/api/v1/characters/"+characterID+"/position",
		bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: UpdatePosition: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
	return nil
}

// DeductGold atomically deducts amount from character's gold balance.
// Returns ErrInsufficientGold if the character doesn't have enough gold.
// Calls PATCH /api/v1/characters/:id/gold.
func (c *Client) DeductGold(characterID string, amount int) error {
	body, _ := json.Marshal(map[string]interface{}{"deduct": amount})
	req, _ := http.NewRequest(http.MethodPatch,
		c.baseURL+"/api/v1/characters/"+characterID+"/gold",
		bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: DeductGold: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusConflict:
		return ErrInsufficientGold
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
}

// TravelTelecrystal validates gold, deducts cost, and moves character to target scene/pos.
// This is an idempotent compound operation: GetCharacter → DeductGold → UpdatePosition.
func (c *Client) TravelTelecrystal(characterID string, castCost, targetScene int, tx, ty, tz float64) error {
	ch, err := c.GetCharacter(characterID)
	if err != nil {
		return err
	}
	if ch.GoldBalance < castCost {
		return ErrInsufficientGold
	}
	if err := c.DeductGold(characterID, castCost); err != nil {
		return err
	}
	return c.UpdatePosition(characterID, targetScene, tx, ty, tz)
}
