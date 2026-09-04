// Package modevent implements GFD's real mod interface for an event broker (GFD-x-123/GFD-x-124,
// founder: "mod interface for event broker in server mods should be able to register and or
// subscribe to specific named events in the system (USE PARENA TYPES) mods should fire off
// signals for their callbacks" + "mods can subscribe to certain events provided by core mods or
// mod mods and can fire off callbacks on specific event types events can have generic payload
// must be typed").
//
// Real, checked-first finding: apps2/mud (GFD's Go server) has had ZERO PARENA-mod integration
// point until this package -- apps2/battlegrounds_gui (the C client) already has one
// (GFD-MACRO-0012's action_bar_mod, compiled via the real `parena build` C target), but a Go
// host needs BURROW's own Go emission target instead (`burrow build ... -o x.go`), which real,
// checked at BURROW/README.md's own "Status," has been shipped since 2026-08-30 but never
// actually consumed by a real host anywhere in this monorepo until this package. This is that
// real, first dogfooding case.
//
// Real design, matching "generic payload... must be typed": events are identified by a plain
// string Name (so any future core code -- or a mod itself, "mods should fire off signals" -- can
// define a new named event without a new Go type), and every Handler is int32-in/int32-out --
// the same real I32-only ceiling every PARENA/BURROW-emitted function in this monorepo already
// respects (no F32/struct/Vec across a mod boundary, ECOWAR's card_effect_mod.prn and GFD's own
// action_bar_mod.prn both document the same real constraint). A payload richer than one int32
// (e.g. multiple flags) is the caller's own job to pack/unpack -- this package doesn't grow a
// per-event payload type, it stays one real, generic, typed shape for every event.
package modevent

import "sync"

// Handler is a real, typed subscriber callback -- in practice, a BURROW-compiled PARENA mod
// function's own generated Go signature (see internal/burrowgen/*.go for real examples), but any
// plain Go func matching this shape works too (a mod-authoring on-ramp doesn't require every
// caller to have gone through PARENA -- same "mod is real code, not a special format" precedent
// GFD-MACRO-0012's own action_bar_mod_host.h already established).
type Handler func(payload int32) int32

// Broker is the real, generic pub/sub registry. Safe for concurrent Subscribe/Publish.
type Broker struct {
	mu   sync.RWMutex
	subs map[string][]Handler
}

// NewBroker constructs an empty broker.
func NewBroker() *Broker {
	return &Broker{subs: make(map[string][]Handler)}
}

// Subscribe registers h to be called every time name is Published. Multiple handlers can
// subscribe to the same name -- Publish calls all of them, in registration order.
func (b *Broker) Subscribe(name string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[name] = append(b.subs[name], h)
}

// Publish fires name with payload, returning every subscribed handler's real result in
// registration order. A name with no subscribers returns an empty (not nil-panicking) slice --
// real, honest fail-open: core code publishing an event nobody has wired a mod to yet is not an
// error, exactly like GFD-MOBSPAWN-001's own spawn.Registry.Enabled fail-open default.
func (b *Broker) Publish(name string, payload int32) []int32 {
	b.mu.RLock()
	handlers := append([]Handler(nil), b.subs[name]...) // snapshot: don't hold the lock during callbacks
	b.mu.RUnlock()
	if len(handlers) == 0 {
		return nil
	}
	out := make([]int32, len(handlers))
	for i, h := range handlers {
		out[i] = h(payload)
	}
	return out
}

// SubscriberCount reports how many handlers are registered for name (0 if none) -- useful for
// tests and for a future admin page to show what's actually wired, matching this monorepo's own
// "the admin GUI needs a real read path" precedent (GFD-NM-124).
func (b *Broker) SubscriberCount(name string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs[name])
}
