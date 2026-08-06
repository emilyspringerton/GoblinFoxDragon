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
	"sync"
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
	HomeSceneID int     `json:"home_scene_id"`
	HomePosX    float64 `json:"home_pos_x"`
	HomePosY    float64 `json:"home_pos_y"`
	HomePosZ    float64 `json:"home_pos_z"`
}

// Client calls the IDUNA MMO API.
type Client struct {
	baseURL     string
	agentName   string
	agentSecret string
	http        *http.Client

	// jwt/jwtExpiry cache the real signed JWT obtained from POST /api/v1/auth/agent -- see
	// ensureToken's own doc comment for why this exists (a real, previously-undiscovered bug:
	// every call this package makes used to send agentSecret itself as the Bearer token, which
	// IDUNA's RequireAuth middleware rejects outright since it isn't a signed JWT).
	mu        sync.Mutex
	jwt       string
	jwtExpiry time.Time
}

// New returns a Client reading IDUNA_BASE_URL, IDUNA_AGENT_NAME, and IDUNA_AGENT_SECRET from env.
func New() *Client {
	base := os.Getenv("IDUNA_BASE_URL")
	if base == "" {
		base = "http://localhost:8080"
	}
	return &Client{
		baseURL:     base,
		agentName:   os.Getenv("IDUNA_AGENT_NAME"),
		agentSecret: os.Getenv("IDUNA_AGENT_SECRET"),
		http:        &http.Client{Timeout: 5 * time.Second},
	}
}

// jwtRefreshMargin is how far ahead of a cached JWT's real expiry (1 hour, IDUNA's own
// AgentAuthHandler) ensureToken treats it as due for renewal -- avoids a request racing an
// expiry that lands mid-flight.
const jwtRefreshMargin = 60 * time.Second

// ensureToken performs the real POST /api/v1/auth/agent login exchange and caches the resulting
// JWT, refreshing it once it's within jwtRefreshMargin of its real 1-hour expiry. This was
// missing entirely before (2026-07-31, found while wiring REDGARDEN_GUI_NORTHSTAR.md Milestone
// 3): do() used to send agentSecret directly as the Bearer token, which IDUNA's real
// jwt.Verify-based RequireAuth middleware has always rejected with 401 -- every call this
// package has ever made (GetCharacter/CreateCharacter/CreditGold/etc., from both apps2/mud and
// apps2/server-go, which share this package) has been silently failing, masked by "best-effort,
// non-blocking" error handling at every call site. Confirmed live against the running IDUNA
// service, not just theorized from reading the code.
func (c *Client) ensureToken() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.jwt != "" && time.Now().Before(c.jwtExpiry.Add(-jwtRefreshMargin)) {
		return nil
	}
	if c.agentName == "" || c.agentSecret == "" {
		return fmt.Errorf("idunaclient: IDUNA_AGENT_NAME/IDUNA_AGENT_SECRET not set")
	}
	body, _ := json.Marshal(map[string]string{"agent_name": c.agentName, "agent_secret": c.agentSecret})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/auth/agent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: agent login: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("idunaclient: agent login: status %d", resp.StatusCode)
	}
	var res struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return fmt.Errorf("idunaclient: agent login decode: %w", err)
	}
	c.jwt = res.AccessToken
	c.jwtExpiry = time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)
	return nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.agentSecret != "" {
		if err := c.ensureToken(); err != nil {
			return nil, err
		}
		c.mu.Lock()
		token := c.jwt
		c.mu.Unlock()
		req.Header.Set("Authorization", "Bearer "+token)
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

// UpdateHome patches the character's real, persisted Home Point (2026-08-04, founder: "iterate" --
// real gap found live earlier the same day: sethome only ever mutated apps2/mud's own in-memory
// homePoint struct, never IDUNA, so a custom Home Point silently reverted to unset on every fresh
// session. Same shape as UpdatePosition.
func (c *Client) UpdateHome(characterID string, sceneID int, x, y, z float64) error {
	body, _ := json.Marshal(map[string]interface{}{
		"scene_id": sceneID, "pos_x": x, "pos_y": y, "pos_z": z,
	})
	req, _ := http.NewRequest(http.MethodPatch,
		c.baseURL+"/api/v1/characters/"+characterID+"/home",
		bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: UpdateHome: %w", err)
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

// UpdateJob patches the character's real, persisted job_main/job_sub (2026-08-05, real gap found
// diagnosing "1/2/3 ability hotkeys don't match my real spells in Meadow"): cmdSetJob only ever
// mutated apps2/mud's own in-memory p.jobID, never IDUNA, so any fresh session (relaunch,
// reconnect) silently reverted the ability bar to whatever job the character was created with.
// Same shape as UpdateHome. jobSub may be empty to clear it, matching cmdSetSubJob's own "NONE".
func (c *Client) UpdateJob(characterID, jobMain, jobSub string) error {
	body, _ := json.Marshal(map[string]interface{}{
		"job_main": jobMain, "job_sub": jobSub,
	})
	req, _ := http.NewRequest(http.MethodPatch,
		c.baseURL+"/api/v1/characters/"+characterID+"/job",
		bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: UpdateJob: %w", err)
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

// CreditGold adds gold to a character's balance -- the symmetric counterpart DeductGold never
// had (IDUNA/internal/http/handlers/mmo.go's own new handleCreditGold, 2026-07-31, EMILY/
// BACKLOG.md "unify the backends" follow-up). IDUNA bounds a single credit call (currently
// 10,000) as a sanity cap against unbounded minting; ErrServer wraps that rejection same as any
// other non-2xx/404 response, callers don't need a distinct case for it.
func (c *Client) CreditGold(characterID string, amount int) error {
	body, _ := json.Marshal(map[string]interface{}{"credit": amount})
	req, _ := http.NewRequest(http.MethodPatch,
		c.baseURL+"/api/v1/characters/"+characterID+"/gold/credit",
		bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: CreditGold: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
}

// BattlegroundsTicket is the response from POST /api/v1/redgarden/player-ticket.
type BattlegroundsTicket struct {
	Ticket    string `json:"ticket"`     // hex-encoded, hand to the REDGARDEN client's --ticket flag
	ExpiresAt int64  `json:"expires_at"` // unix seconds
	PlayerID  string `json:"player_id"`
}

// MintBattlegroundsTicket mints a real REDGARDEN connect ticket for a real DragonsNShit
// character's own player_id -- REDGARDEN_GUI_NORTHSTAR.md Milestone 3, the Battlegrounds entry
// point (`battlegrounds`/`bg` command in apps2/mud). Requires the DRAGONSNSHIT-MUD agent's
// redgarden.player-ticket.mint permission (IDUNA_AGENT_NAME/IDUNA_AGENT_SECRET env), a
// deliberately separate grant from REDGARDEN-BOTS' own redgarden.ticket.mint -- see
// IDUNA/internal/http/handlers/redgarden_player_ticket.go's own doc comment for the trust model.
func (c *Client) MintBattlegroundsTicket(playerID string) (*BattlegroundsTicket, error) {
	body, _ := json.Marshal(map[string]string{"player_id": playerID})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/redgarden/player-ticket", bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("idunaclient: MintBattlegroundsTicket: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
	var t BattlegroundsTicket
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		return nil, fmt.Errorf("idunaclient: MintBattlegroundsTicket decode: %w", err)
	}
	return &t, nil
}

// Item is one row from GET /api/v1/characters/:id/items.
type Item struct {
	ItemID   string `json:"item_id"`
	ItemType string `json:"item_type"`
	Name     string `json:"name"`
	IL       int    `json:"item_level"`
	Quantity int    `json:"quantity"`
}

// ListItems returns all non-destroyed items owned by characterID.
func (c *Client) ListItems(characterID string) ([]Item, error) {
	req, _ := http.NewRequest(http.MethodGet,
		c.baseURL+"/api/v1/characters/"+characterID+"/items", nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("idunaclient: ListItems: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
	var body struct {
		Items []Item `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("idunaclient: ListItems decode: %w", err)
	}
	return body.Items, nil
}

// CreateItem calls POST /api/v1/items to record a newly crafted item.
// Returns the new item_id.
func (c *Client) CreateItem(ownerID, crafterID, itemType, name string, il int) (string, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"owner_character_id": ownerID,
		"crafter_id":         crafterID,
		"item_type":          itemType,
		"name":               name,
		"item_level":         il,
		"quantity":           1,
	})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/items", bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("idunaclient: CreateItem: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
	var result struct {
		ItemID string `json:"item_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("idunaclient: CreateItem decode: %w", err)
	}
	return result.ItemID, nil
}

// DestroyItem calls DELETE /api/v1/items/:id (soft-delete).
func (c *Client) DestroyItem(itemID string) error {
	req, _ := http.NewRequest(http.MethodDelete, c.baseURL+"/api/v1/items/"+itemID, nil)
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: DestroyItem: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
}

// IncrementSkill adds delta to character's skill_name, capped at 110.0.
func (c *Client) IncrementSkill(characterID, skillName string, delta float64) error {
	body, _ := json.Marshal(map[string]interface{}{
		"skill_name": skillName,
		"delta":      delta,
	})
	req, _ := http.NewRequest(http.MethodPatch,
		c.baseURL+"/api/v1/characters/"+characterID+"/skills",
		bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: IncrementSkill: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
}

// PatchWorldEvent patches a world event's phase and ley_integrity in IDUNA.
func (c *Client) PatchWorldEvent(eventID, phase string, leyIntegrity int) error {
	body, _ := json.Marshal(map[string]interface{}{
		"phase":         phase,
		"ley_integrity": leyIntegrity,
	})
	req, _ := http.NewRequest(http.MethodPatch,
		c.baseURL+"/api/v1/world-events/"+eventID,
		bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: PatchWorldEvent: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusNotFound:
		return ErrNotFound
	default:
		return fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
}

// CreateCharacter creates a new character record in IDUNA. Returns the new character_id.
func (c *Client) CreateCharacter(playerID, name, jobMain string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"player_id": playerID,
		"name":      name,
		"job_main":  jobMain,
	})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/characters", bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return "", fmt.Errorf("idunaclient: CreateCharacter: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return "", ErrConflict
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
	var res struct {
		CharacterID string `json:"character_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("idunaclient: CreateCharacter decode: %w", err)
	}
	return res.CharacterID, nil
}

// UpdateCharacterLevel patches a character's level and current_xp.
//
// Real bug found live 2026-08-02 (GoblinFoxDragon's Town headless-combat work): this called
// plain PATCH /api/v1/characters/:id for its entire history -- a route IDUNA has never had (only
// /position, /gold, /gold/credit, /skills exist). Every caller (this package is shared by
// apps2/mud and apps2/server-go) has had every level/XP persist silently fail, masked by
// "best-effort" error handling everywhere it's called from. Fixed to hit the real new
// /api/v1/characters/:id/level route (IDUNA's own mmo.go handleUpdateLevel).
func (c *Client) UpdateCharacterLevel(characterID string, level, currentXP int) error {
	body, _ := json.Marshal(map[string]any{
		"level":      level,
		"current_xp": currentXP,
	})
	req, _ := http.NewRequest(http.MethodPatch, c.baseURL+"/api/v1/characters/"+characterID+"/level", bytes.NewReader(body))
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: UpdateCharacterLevel: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
	return nil
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

// ChatMessage is one row from GET /api/v1/chat/messages.
type ChatMessage struct {
	ID           int64  `json:"id"`
	Channel      string `json:"channel"`
	SenderName   string `json:"sender_name"`
	SenderSource string `json:"sender_source"`
	Body         string `json:"body"`
	CreatedAt    string `json:"created_at"`
}

// PostChatMessage relays one chat line to IDUNA's chat_messages relay (2026-08-02,
// REDGARDEN_GUI_NORTHSTAR.md's in-match MUD chat) so REDGARDEN's Battlegrounds GUI client can
// pick it up. channel is "say"|"yell"|"guild"|"battlegrounds"; senderSource here is always
// "mud" -- the Battlegrounds client posts its own messages with senderSource "battlegrounds"
// directly. Thin wrapper over PostChatMessageAs kept for apps2/mud's existing call site.
func (c *Client) PostChatMessage(channel, senderName, body string) error {
	return c.PostChatMessageAs(channel, senderName, "mud", body)
}

// PostChatMessageAs is PostChatMessage with an explicit senderSource -- added for S171-04
// (apps2/server-go posting as "gfd_server" to the GFD<->EINHORN_SURVIVAL bridge, which reuses
// this same real endpoint rather than a parallel one; see GoblinFoxDragon/docs2/
// CHAT_BRIDGE_TO_EINHORN_SURVIVAL_SPEC.md). IDUNA's own validChatSources gates which values are
// actually accepted -- passing an unregistered source fails server-side, not silently.
func (c *Client) PostChatMessageAs(channel, senderName, senderSource, body string) error {
	reqBody, _ := json.Marshal(map[string]string{
		"channel": channel, "sender_name": senderName, "sender_source": senderSource, "body": body,
	})
	req, _ := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/chat/messages", bytes.NewReader(reqBody))
	resp, err := c.do(req)
	if err != nil {
		return fmt.Errorf("idunaclient: PostChatMessageAs: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
	return nil
}

// GetChatMessages polls for chat messages with id > sinceID, oldest first, capped at limit.
func (c *Client) GetChatMessages(sinceID int64, limit int) ([]ChatMessage, error) {
	url := fmt.Sprintf("%s/api/v1/chat/messages?since_id=%d&limit=%d", c.baseURL, sinceID, limit)
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	resp, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("idunaclient: GetChatMessages: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrServer, resp.StatusCode)
	}
	var msgs []ChatMessage
	if err := json.NewDecoder(resp.Body).Decode(&msgs); err != nil {
		return nil, fmt.Errorf("idunaclient: GetChatMessages decode: %w", err)
	}
	return msgs, nil
}
