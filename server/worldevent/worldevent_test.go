package worldevent_test

import (
	"strings"
	"testing"
	"time"

	"dragonsnshit/server/worldevent"
)

func TestPostAndCount(t *testing.T) {
	r := worldevent.NewRegistry()
	r.Post(worldevent.Event{Type: worldevent.FactionWarStart, Message: "war begins"})
	if r.Count() != 1 {
		t.Errorf("want 1, got %d", r.Count())
	}
}

func TestPostSetsTimestamp(t *testing.T) {
	r := worldevent.NewRegistry()
	before := time.Now().UTC()
	r.Post(worldevent.Event{Type: worldevent.DragonSighting, Message: "dragon seen"})
	after := time.Now().UTC()
	rec := r.Recent(1)
	if len(rec) != 1 {
		t.Fatal("expected 1 event")
	}
	ts := rec[0].OccurredAt
	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp out of range")
	}
}

func TestPostPreservesProvidedTimestamp(t *testing.T) {
	r := worldevent.NewRegistry()
	custom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	r.Post(worldevent.Event{Type: worldevent.MythSeeded, Message: "myth", OccurredAt: custom})
	rec := r.Recent(1)
	if !rec[0].OccurredAt.Equal(custom) {
		t.Errorf("timestamp not preserved")
	}
}

func TestSubscribeReceivesEvents(t *testing.T) {
	r := worldevent.NewRegistry()
	var received []worldevent.Event
	r.Subscribe(func(e worldevent.Event) {
		received = append(received, e)
	})
	r.Post(worldevent.Event{Type: worldevent.RogueSwarmEmergence, Message: "swarm"})
	if len(received) != 1 {
		t.Fatalf("want 1 event, got %d", len(received))
	}
	if received[0].Type != worldevent.RogueSwarmEmergence {
		t.Errorf("type mismatch")
	}
}

func TestMultipleSubscribers(t *testing.T) {
	r := worldevent.NewRegistry()
	var a, b int
	r.Subscribe(func(worldevent.Event) { a++ })
	r.Subscribe(func(worldevent.Event) { b++ })
	r.Post(worldevent.Event{Type: worldevent.EmilyCast, Message: "Emily speaks"})
	if a != 1 || b != 1 {
		t.Errorf("want both handlers called once, got a=%d b=%d", a, b)
	}
}

func TestRecentLimit(t *testing.T) {
	r := worldevent.NewRegistry()
	for i := 0; i < 5; i++ {
		r.Post(worldevent.Event{Type: worldevent.FieldOfficeFallen, Message: "fo fell"})
	}
	rec := r.Recent(3)
	if len(rec) != 3 {
		t.Errorf("want 3, got %d", len(rec))
	}
}

func TestRecentAll(t *testing.T) {
	r := worldevent.NewRegistry()
	r.Post(worldevent.Event{Type: worldevent.FactionWarStart, Message: "m1"})
	r.Post(worldevent.Event{Type: worldevent.DragonSighting, Message: "m2"})
	rec := r.Recent(100)
	if len(rec) != 2 {
		t.Errorf("want 2, got %d", len(rec))
	}
}

func TestRecentEmpty(t *testing.T) {
	r := worldevent.NewRegistry()
	if rec := r.Recent(5); rec != nil {
		t.Errorf("expected nil for empty registry, got %v", rec)
	}
}

func TestRecentZero(t *testing.T) {
	r := worldevent.NewRegistry()
	r.Post(worldevent.Event{Type: worldevent.MythSeeded, Message: "m"})
	if rec := r.Recent(0); rec != nil {
		t.Errorf("expected nil for n=0, got %v", rec)
	}
}

func TestFormatMessageGlobal(t *testing.T) {
	e := worldevent.Event{Type: worldevent.FactionWarStart, Message: "war in district A"}
	msg := worldevent.FormatMessage(e)
	if !strings.Contains(msg, "WORLD") {
		t.Errorf("expected WORLD in format: %s", msg)
	}
	if !strings.Contains(msg, "faction_war_start") {
		t.Errorf("expected event type in format: %s", msg)
	}
}

func TestFormatMessageWithDistrict(t *testing.T) {
	e := worldevent.Event{
		Type:     worldevent.FieldOfficeFallen,
		Message:  "FO-01 fell",
		District: "district-residential",
	}
	msg := worldevent.FormatMessage(e)
	if !strings.Contains(msg, "district-residential") {
		t.Errorf("expected district in format: %s", msg)
	}
}

func TestEventTypes(t *testing.T) {
	types := []worldevent.EventType{
		worldevent.FactionWarStart,
		worldevent.FieldOfficeFallen,
		worldevent.RogueSwarmEmergence,
		worldevent.DragonSighting,
		worldevent.MythSeeded,
		worldevent.EmilyCast,
	}
	for _, et := range types {
		if et == "" {
			t.Errorf("event type should not be empty")
		}
	}
}

func TestCountMultiple(t *testing.T) {
	r := worldevent.NewRegistry()
	for i := 0; i < 10; i++ {
		r.Post(worldevent.Event{Type: worldevent.EmilyCast, Message: "cast"})
	}
	if r.Count() != 10 {
		t.Errorf("want 10, got %d", r.Count())
	}
}
