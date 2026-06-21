package chat

import (
	"strings"
	"testing"
)

func setup3Players() (*Router, string, string, string) {
	r := New()
	r.Register("alice", Session{Name: "Alice", SceneID: 1, GuildID: "guild-A", Pos: Pos{0, 0, 0}})
	r.Register("bob", Session{Name: "Bob", SceneID: 1, GuildID: "guild-A", Pos: Pos{10, 0, 0}})
	r.Register("carol", Session{Name: "Carol", SceneID: 2, GuildID: "guild-B", Pos: Pos{0, 0, 0}})
	return r, "alice", "bob", "carol"
}

// --- say ---

func TestSayReachesNearbyInSameScene(t *testing.T) {
	r, alice, bob, _ := setup3Players()
	dels := r.DeliverFromSlot(alice, ChatSay, "", "hello", 50)
	if !hasRecipient(dels, bob) {
		t.Error("say: Bob (10 units away, same scene) should receive")
	}
}

func TestSayExcludesDifferentScene(t *testing.T) {
	r, alice, _, carol := setup3Players()
	dels := r.DeliverFromSlot(alice, ChatSay, "", "hello", 50)
	if hasRecipient(dels, carol) {
		t.Error("say: Carol (different scene) should NOT receive")
	}
}

func TestSayExcludesOutOfRange(t *testing.T) {
	r := New()
	r.Register("a", Session{Name: "A", SceneID: 1, Pos: Pos{0, 0, 0}})
	r.Register("b", Session{Name: "B", SceneID: 1, Pos: Pos{200, 0, 0}}) // 200 > 50
	dels := r.DeliverFromSlot("a", ChatSay, "", "hi", 50)
	if hasRecipient(dels, "b") {
		t.Error("say: player 200 units away should NOT receive")
	}
}

func TestSayIncludesSelf(t *testing.T) {
	r, alice, _, _ := setup3Players()
	dels := r.DeliverFromSlot(alice, ChatSay, "", "hi", 50)
	if !hasRecipient(dels, alice) {
		t.Error("say: sender should receive own message (echo)")
	}
}

// --- tell ---

func TestTellReachesTarget(t *testing.T) {
	r, alice, _, carol := setup3Players()
	// carol is in a different scene — tell still works
	dels := r.DeliverFromSlot(alice, ChatTell, "Carol", "psst", 50)
	if !hasRecipient(dels, carol) {
		t.Error("tell: Carol should receive private message")
	}
}

func TestTellExcludesOthers(t *testing.T) {
	r, alice, bob, carol := setup3Players()
	dels := r.DeliverFromSlot(alice, ChatTell, "Carol", "psst", 50)
	if hasRecipient(dels, bob) {
		t.Error("tell: Bob should NOT receive Carol's private message")
	}
	_ = carol
}

func TestTellUnknownTargetReturnsNil(t *testing.T) {
	r, alice, _, _ := setup3Players()
	dels := r.DeliverFromSlot(alice, ChatTell, "Nobody", "hello", 50)
	if len(dels) != 0 {
		t.Errorf("tell to unknown target: want 0 deliveries, got %d", len(dels))
	}
}

// --- yell ---

func TestYellReachesEntireScene(t *testing.T) {
	r, alice, bob, carol := setup3Players()
	r.Register("dave", Session{Name: "Dave", SceneID: 1, Pos: Pos{500, 0, 0}}) // far but same scene

	dels := r.DeliverFromSlot(alice, ChatYell, "", "HELLO ZONE", 50)
	if !hasRecipient(dels, bob) {
		t.Error("yell: Bob (same scene) should receive")
	}
	if !hasRecipient(dels, "dave") {
		t.Error("yell: Dave (far, same scene) should receive")
	}
	if hasRecipient(dels, carol) {
		t.Error("yell: Carol (different scene) should NOT receive")
	}
}

// --- guild ---

func TestGuildChatReachesGuildMembers(t *testing.T) {
	r, alice, bob, carol := setup3Players()
	// alice and bob are guild-A; carol is guild-B
	dels := r.DeliverFromSlot(alice, ChatGuild, "", "guild only", 50)
	if !hasRecipient(dels, bob) {
		t.Error("guild: Bob (same guild) should receive")
	}
	if hasRecipient(dels, carol) {
		t.Error("guild: Carol (different guild) should NOT receive")
	}
}

func TestGuildChatCrossScene(t *testing.T) {
	r := New()
	r.Register("a", Session{Name: "A", SceneID: 1, GuildID: "g1"})
	r.Register("b", Session{Name: "B", SceneID: 2, GuildID: "g1"}) // different scene, same guild
	dels := r.DeliverFromSlot("a", ChatGuild, "", "across scenes", 50)
	if !hasRecipient(dels, "b") {
		t.Error("guild: member in different scene should still receive guild chat")
	}
}

func TestGuildChatNoGuildReturnsNil(t *testing.T) {
	r := New()
	r.Register("a", Session{Name: "A", SceneID: 1, GuildID: ""})
	dels := r.DeliverFromSlot("a", ChatGuild, "", "hello guild?", 50)
	if len(dels) != 0 {
		t.Errorf("guild chat from unguilded player: want 0 deliveries, got %d", len(dels))
	}
}

// --- packet encoding ---

func TestEncodeChatStructure(t *testing.T) {
	pkt := encodeChat(ChatSay, "Alice", "hello")
	if pkt[0] != 6 {
		t.Errorf("packet type: got %d, want 6", pkt[0])
	}
	if pkt[1] != ChatSay {
		t.Errorf("channel: got %d, want %d", pkt[1], ChatSay)
	}
	nameLen := int(pkt[2])
	if nameLen != 5 {
		t.Errorf("sender name len: got %d, want 5", nameLen)
	}
	if string(pkt[3:3+nameLen]) != "Alice" {
		t.Errorf("sender name: got %q", pkt[3:3+nameLen])
	}
	msgLen := int(pkt[3+nameLen])
	if string(pkt[4+nameLen:4+nameLen+msgLen]) != "hello" {
		t.Errorf("message: got %q", pkt[4+nameLen:])
	}
}

func TestEncodeChatTruncatesLongName(t *testing.T) {
	name := strings.Repeat("x", 50)
	pkt := encodeChat(ChatSay, name, "hi")
	if pkt[2] != 32 {
		t.Errorf("long name should truncate to 32, got %d", pkt[2])
	}
}

func TestEncodeChatTruncatesLongMsg(t *testing.T) {
	msg := strings.Repeat("m", 250)
	pkt := encodeChat(ChatSay, "A", msg)
	nameLen := int(pkt[2])
	msgLen := int(pkt[3+nameLen])
	if msgLen != 200 {
		t.Errorf("long message should truncate to 200, got %d", msgLen)
	}
}

// --- parse ---

func TestParseChatPacketSay(t *testing.T) {
	// build a client→server say packet
	msg := "hi there"
	buf := []byte{6, ChatSay, 0, byte(len(msg))}
	buf = append(buf, []byte(msg)...)
	ch, target, got, ok := ParseChatPacket(buf)
	if !ok {
		t.Fatal("ParseChatPacket: not ok")
	}
	if ch != ChatSay {
		t.Errorf("channel: got %d, want %d", ch, ChatSay)
	}
	if target != "" {
		t.Errorf("target: got %q, want empty", target)
	}
	if got != msg {
		t.Errorf("msg: got %q, want %q", got, msg)
	}
}

func TestParseChatPacketTell(t *testing.T) {
	target := "Bob"
	msg := "hey"
	buf := []byte{6, ChatTell, byte(len(target))}
	buf = append(buf, []byte(target)...)
	buf = append(buf, byte(len(msg)))
	buf = append(buf, []byte(msg)...)
	ch, gotTarget, gotMsg, ok := ParseChatPacket(buf)
	if !ok {
		t.Fatal("ParseChatPacket tell: not ok")
	}
	if ch != ChatTell {
		t.Errorf("channel: got %d", ch)
	}
	if gotTarget != "Bob" {
		t.Errorf("target: got %q", gotTarget)
	}
	if gotMsg != msg {
		t.Errorf("msg: got %q", gotMsg)
	}
}

func TestParseChatPacketTooShort(t *testing.T) {
	_, _, _, ok := ParseChatPacket([]byte{6, 0})
	if ok {
		t.Error("should return ok=false for too-short packet")
	}
}

// --- misc ---

func TestEmptyMessageReturnsNil(t *testing.T) {
	r, alice, _, _ := setup3Players()
	dels := r.DeliverFromSlot(alice, ChatSay, "", "", 50)
	if len(dels) != 0 {
		t.Error("empty message should produce no deliveries")
	}
}

func TestUnregisterRemovesSession(t *testing.T) {
	r, alice, bob, _ := setup3Players()
	r.Unregister(bob)
	dels := r.DeliverFromSlot(alice, ChatYell, "", "anyone?", 50)
	if hasRecipient(dels, bob) {
		t.Error("unregistered slot should not receive messages")
	}
}

// --- helpers ---

func hasRecipient(dels []Delivery, slot string) bool {
	for _, d := range dels {
		if d.To == slot {
			return true
		}
	}
	return false
}
