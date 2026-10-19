package mob

import (
	"fmt"
	"strings"
	"time"
	"unicode"
)

// Multi-room dungeons. Each roster dungeon is a chain of six room zones (see
// scripts/gen_dungeons.py): an empty entrance, two corridors, a side chamber with an elite, an
// antechamber with a second elite and a boss hall. The tables in dungeon_rooms_gen.go give each
// room's mobs; the boss and elites come from DungeonRoster at spawn time, so the IDUNA dungeon
// roster editor (data/dungeon_roster.json) still decides who the heroes are.

// DungeonRoomMob is one group of mobs in a room.
type DungeonRoomMob struct {
	Role       string // "minion", "elite" or "boss"
	Kind       string // minion kind (elite/boss kinds come from the roster)
	EliteIdx   int    // which roster elite (wraps)
	HP, Damage int
	Count      int
	AggroRange float64
	MoveSpeed  float64
}

// DungeonRoom is one room zone.
type DungeonRoom struct {
	ZoneID  int
	Dungeon int
	Index   int // 0 = entrance
	Name    string
	Level   int
	Mobs    []DungeonRoomMob
}

// HeroName is the player-facing character name of a roster hero id (title-cased fallback for a hero
// the table does not know, e.g. one added later through the roster editor).
func HeroName(heroID string) string {
	if n, ok := DungeonHeroNames[heroID]; ok {
		return n
	}
	words := strings.Split(strings.ToLower(strings.TrimPrefix(heroID, "ARENA_HERO_")), "_")
	for i, w := range words {
		if w != "" {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// DungeonRoomZoneIDs returns the zone ids of every room that is its own zone (rooms 1-5 of each
// dungeon; room 0 is the roster dungeon's own zone, 208+i).
func DungeonRoomZoneIDs() []int {
	var ids []int
	for _, r := range DungeonRoomTable {
		if r.Index > 0 {
			ids = append(ids, r.ZoneID)
		}
	}
	return ids
}

// DungeonRoomFor returns the room for a zone id.
func DungeonRoomFor(zoneID int) (DungeonRoom, bool) {
	for _, r := range DungeonRoomTable {
		if r.ZoneID == zoneID {
			return r, true
		}
	}
	return DungeonRoom{}, false
}

func idSlug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// DungeonRoomSpawns builds the mobs of one room zone.
func DungeonRoomSpawns(zoneID int) []Mob {
	room, ok := DungeonRoomFor(zoneID)
	if !ok {
		return nil
	}
	var roster DungeonBossAssignment
	if room.Dungeon >= 0 && room.Dungeon < len(DungeonRoster) {
		roster = DungeonRoster[room.Dungeon]
	}
	var out []Mob
	for gi, g := range room.Mobs {
		kind := g.Kind
		res := 0.0
		switch g.Role {
		case "boss":
			kind = HeroName(roster.Boss)
			if roster.Boss == "" {
				kind = "The Warden"
			}
			res = 0.6
		case "elite":
			if n := len(roster.Elite); n > 0 {
				kind = HeroName(roster.Elite[g.EliteIdx%n])
			} else if room.Dungeon < len(DungeonFallbackElites) {
				kind = DungeonFallbackElites[room.Dungeon]
			}
			res = 0.3
		}
		for n := 0; n < g.Count; n++ {
			// spread the group on a small ring so they do not stack
			angle := float64(n)*2.4 + float64(gi)
			pos := Pos{X: 6 * cosApprox(angle), Y: 1, Z: 6 * sinApprox(angle)}
			if g.Role != "minion" {
				pos = Pos{X: 0, Y: 1, Z: -8}
			}
			id := fmt.Sprintf("%s-d%d-r%d-%d-%d", idSlug(kind), room.Dungeon, room.Index, gi, n)
			out = append(out, Mob{
				ID: id, Kind: kind, SceneID: zoneID, Pos: pos, HomePos: pos,
				HP: g.HP, MaxHP: g.HP, State: StateIdle,
				AggroRange: g.AggroRange, LeashRange: g.AggroRange*2 + 20, MeleeRange: 2.5,
				MoveSpeed: g.MoveSpeed, MeleeDamage: g.Damage, SwingDelay: 2800 * time.Millisecond, CCResist: res,
			})
		}
	}
	return out
}
