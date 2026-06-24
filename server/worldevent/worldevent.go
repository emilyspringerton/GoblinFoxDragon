// Package worldevent implements the TRAPX world event broadcast system.
//
// World events are server-wide notifications that all connected players receive.
// They can originate from game systems (faction wars, rogue swarms, dragon sightings)
// or from Emily Prime via the HTTP POST /api/world-events endpoint.
package worldevent

import (
	"sync"
	"time"
)

// EventType identifies the kind of world event.
type EventType string

const (
	FactionWarStart     EventType = "faction_war_start"
	FieldOfficeFallen   EventType = "field_office_fallen"
	RogueSwarmEmergence EventType = "rogue_swarm_emergence"
	DragonSighting      EventType = "dragon_sighting"
	MythSeeded          EventType = "myth_seeded"
	EmilyCast           EventType = "emily_cast" // Emily Prime external POST
)

// Event is a single world event ready for broadcast.
type Event struct {
	Type      EventType
	Message   string
	District  string    // optional; empty = global
	OccurredAt time.Time
}

// Handler is called when a new event is posted to the registry.
// Implementations should be non-blocking (the Registry holds no lock while calling).
type Handler func(e Event)

// Registry is the central world event bus.
type Registry struct {
	mu       sync.Mutex
	log      []Event
	handlers []Handler
}

// NewRegistry creates an empty world event registry.
func NewRegistry() *Registry { return &Registry{} }

// Subscribe registers a handler that will be called for every future event.
// Thread-safe; may be called concurrently with Post.
func (r *Registry) Subscribe(h Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers = append(r.handlers, h)
}

// Post records and broadcasts a world event.
func (r *Registry) Post(e Event) {
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	r.mu.Lock()
	r.log = append(r.log, e)
	handlers := make([]Handler, len(r.handlers))
	copy(handlers, r.handlers)
	r.mu.Unlock()

	for _, h := range handlers {
		h(e)
	}
}

// Recent returns the n most recent events (newest last).
func (r *Registry) Recent(n int) []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || len(r.log) == 0 {
		return nil
	}
	start := len(r.log) - n
	if start < 0 {
		start = 0
	}
	out := make([]Event, len(r.log)-start)
	copy(out, r.log[start:])
	return out
}

// Count returns the total number of events posted.
func (r *Registry) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.log)
}

// FormatMessage builds the global broadcast string shown to all players.
// Format: "*** [EVENT_TYPE] Message ***" or with district "*** [DISTRICT] EVENT_TYPE: Message ***"
func FormatMessage(e Event) string {
	if e.District != "" {
		return "*** [" + e.District + "] " + string(e.Type) + ": " + e.Message + " ***"
	}
	return "*** [WORLD] " + string(e.Type) + ": " + e.Message + " ***"
}
