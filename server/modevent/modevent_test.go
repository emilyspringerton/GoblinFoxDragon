package modevent

import "testing"

func TestPublish_NoSubscribers_ReturnsNilNotPanic(t *testing.T) {
	b := NewBroker()
	got := b.Publish("nothing.subscribed", 5)
	if got != nil {
		t.Fatalf("expected nil for an unsubscribed event, got %v", got)
	}
}

func TestSubscribe_Publish_CallsHandlerWithPayload(t *testing.T) {
	b := NewBroker()
	var gotPayload int32
	b.Subscribe("mob.death", func(payload int32) int32 {
		gotPayload = payload
		return payload * 2
	})
	got := b.Publish("mob.death", 21)
	if gotPayload != 21 {
		t.Fatalf("expected handler to receive payload 21, got %d", gotPayload)
	}
	if len(got) != 1 || got[0] != 42 {
		t.Fatalf("expected [42], got %v", got)
	}
}

func TestSubscribe_MultipleHandlers_AllCalledInOrder(t *testing.T) {
	b := NewBroker()
	var order []int
	b.Subscribe("evt", func(payload int32) int32 { order = append(order, 1); return 1 })
	b.Subscribe("evt", func(payload int32) int32 { order = append(order, 2); return 2 })
	b.Subscribe("evt", func(payload int32) int32 { order = append(order, 3); return 3 })
	got := b.Publish("evt", 0)
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Fatalf("expected [1 2 3], got %v", got)
	}
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("expected handlers called in registration order, got %v", order)
	}
}

func TestPublish_DifferentEventNames_Independent(t *testing.T) {
	b := NewBroker()
	b.Subscribe("a", func(payload int32) int32 { return 100 })
	got := b.Publish("b", 0)
	if got != nil {
		t.Fatalf("expected event 'b' to have no subscribers, got %v", got)
	}
	got = b.Publish("a", 0)
	if len(got) != 1 || got[0] != 100 {
		t.Fatalf("expected [100] for event 'a', got %v", got)
	}
}

func TestSubscriberCount(t *testing.T) {
	b := NewBroker()
	if b.SubscriberCount("mob.death") != 0 {
		t.Fatal("expected 0 subscribers before any Subscribe call")
	}
	b.Subscribe("mob.death", func(payload int32) int32 { return 0 })
	b.Subscribe("mob.death", func(payload int32) int32 { return 0 })
	if b.SubscriberCount("mob.death") != 2 {
		t.Fatalf("expected 2 subscribers, got %d", b.SubscriberCount("mob.death"))
	}
	if b.SubscriberCount("other.event") != 0 {
		t.Fatal("expected 0 subscribers for a different, never-subscribed event")
	}
}
