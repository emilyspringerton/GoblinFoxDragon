// Package guild implements the DragonsNShit linkshell-style guild system.
//
// # Linkshell model (FFXI-inspired)
//
// A Guild has two in-world item types:
//   - Feather      — grants bearer access to the guild's chat channel (ChatGuild).
//   - Feather Sack — held by the guild leader / officers; used to forge and revoke Feathers.
//
// Only a player holding a Feather Sack can issue Feathers. A player holding a Feather
// can speak and hear the guild channel. Dropping / destroying your Feather removes you
// from the guild. This mirrors FFXI's linkshell/pearlsack mechanic.
//
// The Registry is pure (no I/O, no network). Callers wire it to IDUNA persistence.
package guild

import (
	"errors"
	"fmt"
	"time"
)

// ItemKind distinguishes the two linkshell item types.
type ItemKind uint8

const (
	KindFeather    ItemKind = 1 // member access token
	KindFeatherSack ItemKind = 2 // admin item — can forge/revoke Feathers
)

// Item is a physical in-world guild membership token.
type Item struct {
	ID        string
	GuildID   string
	Kind      ItemKind
	OwnerID   string    // character who holds it; "" = unassigned
	ForgedAt  time.Time
	ForgedBy  string    // character ID of the Sack holder who made it
	RevokedAt time.Time // zero = active
}

// IsActive returns true if the item has not been revoked.
func (it *Item) IsActive() bool { return it.RevokedAt.IsZero() }

// Role within a guild.
type Role uint8

const (
	RoleMember  Role = 1
	RoleOfficer Role = 2
	RoleLeader  Role = 3
)

// Membership links a character to a guild via their Feather item.
type Membership struct {
	CharacterID string
	GuildID     string
	Role        Role
	FeatherID   string    // the Item.ID they hold
	JoinedAt    time.Time
}

// Guild holds guild metadata and its membership roster.
type Guild struct {
	ID          string
	Name        string
	Tag         string // short tag, e.g. "[DNS]"
	FounderID   string
	FoundedAt   time.Time
	members     map[string]*Membership // characterID → Membership
	items       map[string]*Item       // itemID → Item
	nextItemSeq int
}

func newGuild(id, name, tag, founderID string) *Guild {
	return &Guild{
		ID:        id,
		Name:      name,
		Tag:       tag,
		FounderID: founderID,
		FoundedAt: time.Now(),
		members:   make(map[string]*Membership),
		items:     make(map[string]*Item),
	}
}

// Members returns a snapshot of all current memberships.
func (g *Guild) Members() []Membership {
	out := make([]Membership, 0, len(g.members))
	for _, m := range g.members {
		out = append(out, *m)
	}
	return out
}

// Items returns a snapshot of all guild items (active and revoked).
func (g *Guild) Items() []Item {
	out := make([]Item, 0, len(g.items))
	for _, it := range g.items {
		out = append(out, *it)
	}
	return out
}

// Registry is the authoritative in-memory guild store.
// All mutations return errors on policy violations.
// Callers persist state to IDUNA after successful mutations.
type Registry struct {
	guilds map[string]*Guild // guildID → Guild
}

// New returns an empty Registry.
func New() *Registry { return &Registry{guilds: make(map[string]*Guild)} }

var (
	ErrGuildNotFound   = errors.New("guild not found")
	ErrNotMember       = errors.New("not a member of this guild")
	ErrInsufficientRole = errors.New("insufficient role for this action")
	ErrItemNotFound    = errors.New("item not found")
	ErrItemRevoked     = errors.New("item has been revoked")
	ErrAlreadyMember   = errors.New("character is already a member")
	ErrNotItemHolder   = errors.New("character does not hold this item")
)

// CreateGuild founds a new guild and gives the founder a Feather Sack.
// Returns the Guild and the founder's Feather Sack item.
func (r *Registry) CreateGuild(guildID, name, tag, founderID string) (*Guild, *Item, error) {
	g := newGuild(guildID, name, tag, founderID)

	// Founder gets a Feather Sack (leader).
	sack := g.forgeItem(KindFeatherSack, founderID, founderID)
	g.members[founderID] = &Membership{
		CharacterID: founderID,
		GuildID:     guildID,
		Role:        RoleLeader,
		FeatherID:   sack.ID,
		JoinedAt:    time.Now(),
	}
	r.guilds[guildID] = g
	return g, sack, nil
}

// ForgeFeather creates a new Feather and assigns it to recipientID.
// issuerID must hold a Feather Sack in this guild (Officer or Leader).
func (r *Registry) ForgeFeather(guildID, issuerID, recipientID string) (*Item, error) {
	g, err := r.guild(guildID)
	if err != nil {
		return nil, err
	}
	issuer, ok := g.members[issuerID]
	if !ok {
		return nil, ErrNotMember
	}
	if issuer.Role < RoleOfficer {
		return nil, ErrInsufficientRole
	}
	if _, exists := g.members[recipientID]; exists {
		return nil, ErrAlreadyMember
	}
	feather := g.forgeItem(KindFeather, recipientID, issuerID)
	g.members[recipientID] = &Membership{
		CharacterID: recipientID,
		GuildID:     guildID,
		Role:        RoleMember,
		FeatherID:   feather.ID,
		JoinedAt:    time.Now(),
	}
	return feather, nil
}

// ForgeFeatherSack creates a Feather Sack for an existing member, promoting them to Officer.
// issuerID must be Leader.
func (r *Registry) ForgeFeatherSack(guildID, issuerID, recipientID string) (*Item, error) {
	g, err := r.guild(guildID)
	if err != nil {
		return nil, err
	}
	issuer, ok := g.members[issuerID]
	if !ok {
		return nil, ErrNotMember
	}
	if issuer.Role < RoleLeader {
		return nil, ErrInsufficientRole
	}
	recipient, ok := g.members[recipientID]
	if !ok {
		return nil, ErrNotMember
	}
	sack := g.forgeItem(KindFeatherSack, recipientID, issuerID)
	recipient.Role = RoleOfficer
	recipient.FeatherID = sack.ID
	return sack, nil
}

// RevokeItem marks an item revoked and removes the holder from the guild roster.
// issuerID must hold a Feather Sack (Officer+) and must outrank the target.
func (r *Registry) RevokeItem(guildID, issuerID, itemID string) error {
	g, err := r.guild(guildID)
	if err != nil {
		return err
	}
	issuer, ok := g.members[issuerID]
	if !ok {
		return ErrNotMember
	}
	if issuer.Role < RoleOfficer {
		return ErrInsufficientRole
	}
	item, ok := g.items[itemID]
	if !ok {
		return ErrItemNotFound
	}
	if !item.IsActive() {
		return ErrItemRevoked
	}
	// Officers cannot revoke Sacks (leader-only).
	if item.Kind == KindFeatherSack && issuer.Role < RoleLeader {
		return ErrInsufficientRole
	}
	// Cannot revoke issuer's own item.
	if item.OwnerID == issuerID {
		return fmt.Errorf("cannot revoke own item")
	}
	item.RevokedAt = time.Now()
	delete(g.members, item.OwnerID)
	return nil
}

// CanChat returns true if characterID holds an active Feather (or Sack) in guildID.
func (r *Registry) CanChat(guildID, characterID string) bool {
	g, ok := r.guilds[guildID]
	if !ok {
		return false
	}
	m, ok := g.members[characterID]
	if !ok {
		return false
	}
	item, ok := g.items[m.FeatherID]
	if !ok {
		return false
	}
	return item.IsActive()
}

// GuildID returns the guild the character belongs to, or "" if none.
func (r *Registry) GuildID(characterID string) string {
	for id, g := range r.guilds {
		if _, ok := g.members[characterID]; ok {
			return id
		}
	}
	return ""
}

// GetGuild returns a guild by ID.
func (r *Registry) GetGuild(guildID string) (*Guild, error) { return r.guild(guildID) }

// --- internal ---

func (r *Registry) guild(id string) (*Guild, error) {
	g, ok := r.guilds[id]
	if !ok {
		return nil, ErrGuildNotFound
	}
	return g, nil
}

func (g *Guild) forgeItem(kind ItemKind, ownerID, forgedBy string) *Item {
	g.nextItemSeq++
	id := fmt.Sprintf("%s-item-%d", g.ID, g.nextItemSeq)
	it := &Item{
		ID:       id,
		GuildID:  g.ID,
		Kind:     kind,
		OwnerID:  ownerID,
		ForgedAt: time.Now(),
		ForgedBy: forgedBy,
	}
	g.items[id] = it
	return it
}
