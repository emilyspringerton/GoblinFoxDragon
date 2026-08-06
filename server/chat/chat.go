// Package chat implements server-authoritative chat routing for DragonsNShit.
// Four channels: say (local area), tell (private), yell (zone-wide), guild.
// The Router is pure (no I/O); callers supply wire encoding and UDP dispatch.
package chat

import (
	"math"
)

// Pos is a 3-D world position. Mirrors system.Vec3 without the import cycle.
type Pos struct{ X, Y, Z float64 }

func distPos(a, b Pos) float64 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	dz := a.Z - b.Z
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

// Session holds the server's knowledge of a connected client needed for routing.
type Session struct {
	Name    string
	SceneID int
	GuildID string // empty = no guild
	Pos     Pos
}

// Delivery is one outbound message: the recipient slot key and the encoded packet.
type Delivery struct {
	To     string // slot key (e.g. UDP remote.String())
	Packet []byte
}

// Router routes chat messages between connected sessions.
// All methods are safe for single-goroutine use (main server loop).
type Router struct {
	sessions map[string]Session // slot key → session
}

// New returns an empty Router.
func New() *Router { return &Router{sessions: make(map[string]Session)} }

// Register adds or replaces a session. Call on connect / after name/scene/guild change.
func (r *Router) Register(slot string, s Session) { r.sessions[slot] = s }

// Unregister removes a session. Call on disconnect.
func (r *Router) Unregister(slot string) { delete(r.sessions, slot) }

// GetSession returns the session for a slot, and whether it exists.
func (r *Router) GetSession(slot string) (Session, bool) {
	s, ok := r.sessions[slot]
	return s, ok
}

// Deliver routes a chat message from `fromSlot` on the given channel.
// target is only used for ChatTell (recipient name).
// msg must be non-empty and ≤ 200 bytes.
// Returns the list of Deliveries the caller should send over UDP.
func (r *Router) Deliver(fromSlot string, channel int, target, msg string, sayRadius float64) []Delivery {
	if msg == "" || len(msg) > 200 {
		return nil
	}
	sender, ok := r.sessions[fromSlot]
	if !ok {
		return nil
	}

	var recipients []string
	switch channel {
	case ChatSay:
		recipients = r.inRadius(sender.SceneID, sender.Pos, sayRadius)
	case ChatTell:
		if slot, ok := r.findByName(target); ok {
			recipients = []string{slot}
		}
	case ChatYell:
		recipients = r.inScene(sender.SceneID)
	case ChatGuild:
		if sender.GuildID == "" {
			return nil
		}
		recipients = r.inGuild(sender.GuildID)
	default:
		return nil
	}

	if len(recipients) == 0 {
		return nil
	}
	pkt := EncodeChat(channel, sender.Name, msg)
	out := make([]Delivery, 0, len(recipients))
	for _, slot := range recipients {
		out = append(out, Delivery{To: slot, Packet: pkt})
	}
	return out
}

// DeliverFromSlot is an alias for Deliver kept for backward compatibility.
func (r *Router) DeliverFromSlot(fromSlot string, channel int, target, msg string, sayRadius float64) []Delivery {
	return r.Deliver(fromSlot, channel, target, msg, sayRadius)
}

// EncodeChat builds the server→client PacketChat payload.
//
//	[0]   = 6 (PacketChat)
//	[1]   = channel
//	[2]   = sender_name_len
//	[3..] = sender name
//	[3+n] = msg_len (uint8)
//	[4+n..] = message
func EncodeChat(channel int, senderName, msg string) []byte {
	if len(senderName) > 32 {
		senderName = senderName[:32]
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	n := len(senderName)
	m := len(msg)
	buf := make([]byte, 3+n+1+m)
	buf[0] = 6 // PacketChat
	buf[1] = byte(channel)
	buf[2] = byte(n)
	copy(buf[3:], senderName)
	buf[3+n] = byte(m)
	copy(buf[4+n:], msg)
	return buf
}

// ParseChatPacket decodes a client→server PacketChat payload (buf[0] = type byte 6).
//
//	[0] = 6
//	[1] = channel
//	[2] = target_len (0 if not tell)
//	[3..3+target_len-1] = target name
//	[3+target_len] = msg_len
//	[4+target_len..] = message
func ParseChatPacket(buf []byte) (channel int, target, msg string, ok bool) {
	if len(buf) < 4 {
		return
	}
	channel = int(buf[1])
	targetLen := int(buf[2])
	if len(buf) < 3+targetLen+1 {
		return
	}
	if targetLen > 0 {
		target = string(buf[3 : 3+targetLen])
	}
	msgLen := int(buf[3+targetLen])
	if len(buf) < 4+targetLen+msgLen {
		return
	}
	msg = string(buf[4+targetLen : 4+targetLen+msgLen])
	ok = true
	return
}

// --- internal helpers ---

func (r *Router) findByName(name string) (string, bool) {
	for k, v := range r.sessions {
		if v.Name == name {
			return k, true
		}
	}
	return "", false
}

func (r *Router) inScene(sceneID int) []string {
	var out []string
	for k, v := range r.sessions {
		if v.SceneID == sceneID {
			out = append(out, k)
		}
	}
	return out
}

func (r *Router) inRadius(sceneID int, origin Pos, radius float64) []string {
	var out []string
	for k, v := range r.sessions {
		if v.SceneID == sceneID && distPos(v.Pos, origin) <= radius {
			out = append(out, k)
		}
	}
	return out
}

func (r *Router) inGuild(guildID string) []string {
	var out []string
	for k, v := range r.sessions {
		if v.GuildID == guildID {
			out = append(out, k)
		}
	}
	return out
}

// Channel constants.
const (
	ChatSay   = 0
	ChatTell  = 1
	ChatYell  = 2
	ChatGuild = 3
)
