package mob

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"

	"dragonsnshit/server/worldapi"
)

// dungeon.go — DUNGEON_NORTHSTAR.md Milestone 3 ("Mob and boss population from arena heroes...
// Seeded spawn table places hero-kit-driven hostile NPCs per room + a boss in the final room,
// using existing apps/arena_bot AI"). Real, honest, bounded first slice: this builds the real,
// seeded SPAWN TABLE (which room gets which named hero as a minion/elite/boss, and where) using
// this package's own existing Registry.Spawn primitive -- it does NOT drive REDGARDEN's real
// arena_bot AI as a hostile NPC, which the northstar's own §3.3 explicitly names as the real,
// separate, larger cross-repo work ("the real new work is *driving* that existing AI as a
// hostile NPC... not built here"). A Mob's Kind here carries the real hero identifier from §7's
// content pass (e.g. "ARENA_HERO_CART") as an honest placeholder for that future AI hookup --
// today it behaves exactly like any other generic mob (this package's own default aggro/melee
// stats), just tagged with the name/tier a future pass will use to actually wire in that hero's
// kit.

// DungeonBossAssignment mirrors one row of DUNGEON_NORTHSTAR.md §7's real, compendium-grounded
// content pass -- kept here as real Go data (not re-derived) so a seeded spawn table can place
// the actual named roster, not a placeholder. Order matches the northstar's own table exactly.
type DungeonBossAssignment struct {
	Name  string   `json:"name"`  // dungeon name, e.g. "The Sealed Archive"
	Boss  string   `json:"boss"`  // real ARENA_HERO_* identifier
	Elite []string `json:"elite"` // real ARENA_HERO_* identifiers, may be empty
}

// DungeonRoster is DUNGEON_NORTHSTAR.md §7's 8 named dungeons, real and in-order. Kikoryu's
// Hoard (the real superboss/endgame dungeon) is deliberately NOT included here -- §7 itself
// names it as needing genuinely new AI/kit work, not a repurposed arena hero, so it doesn't fit
// this generic "boss = an existing ARENA_HERO_* identifier" table at all.
var DungeonRoster = []DungeonBossAssignment{
	{Name: "The Sealed Archive", Boss: "ARENA_HERO_CART", Elite: []string{"ARENA_HERO_NOOR1"}},
	{Name: "The Frequency Table", Boss: "ARENA_HERO_PAIMON", Elite: []string{"ARENA_HERO_VASSAGO", "ARENA_HERO_BELETH", "ARENA_HERO_ZAGAN", "ARENA_HERO_DOC_WHEEL"}},
	{Name: "The Remnant's Hall", Boss: "ARENA_HERO_GUNNR", Elite: []string{"ARENA_HERO_LOKI", "ARENA_HERO_GARY", "ARENA_HERO_COURIER"}},
	{Name: "East of the Wall", Boss: "ARENA_HERO_HE_XIANGU", Elite: []string{"ARENA_HERO_WEATHERMAN", "ARENA_HERO_FLUTE_DEBT"}},
	{Name: "The Highland Wake", Boss: "ARENA_HERO_MORRIGAN", Elite: []string{"ARENA_HERO_DAGDA"}},
	{Name: "The Founders' Table", Boss: "ARENA_HERO_UNICORN", Elite: []string{"ARENA_HERO_GHOST", "ARENA_HERO_FROG", "ARENA_HERO_TREE", "ARENA_HERO_DUCK"}},
	{Name: "The Unbound Wing", Boss: "ARENA_HERO_CAIN", Elite: []string{"ARENA_HERO_ABRAHAM", "ARENA_HERO_ADA", "ARENA_HERO_MNM"}},
	{Name: "The Proving Grounds", Boss: "ARENA_HERO_WARRIOR", Elite: nil},
}

// LoadDungeonRosterOverride replaces the package-level DungeonRoster from a JSON file
// (GFD-MOBSPAWN-001 Phase 5, same real "hardcoded Go table -> JSON + registry -> GUI" pattern
// already applied to mob drops (server/mobdrop) and spawn toggles (server/spawn) earlier in
// this same effort). Unlike those two, DungeonRoster stays a real, honest compiled-in default
// above -- this is real, compendium-grounded content that must keep working even if
// data/dungeon_roster.json is ever missing or malformed, not a feature that starts out empty.
// The dungeon-order dependency (dungeonIndex selects DungeonRoster[i % len]) is preserved
// automatically since this replaces the whole ordered slice, never merges into it.
func LoadDungeonRosterOverride(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("mob: load dungeon roster override %s: %w", path, err)
	}
	var roster []DungeonBossAssignment
	if err := json.Unmarshal(data, &roster); err != nil {
		return fmt.Errorf("mob: parse dungeon roster override: %w", err)
	}
	if len(roster) == 0 {
		return fmt.Errorf("mob: dungeon roster override %s is empty, keeping the compiled-in default", path)
	}
	DungeonRoster = roster
	return nil
}

const (
	// KindDungeonMinion tags a generic room-filler spawn -- real minion sprite/AI variety is
	// still the founder's own pending art-direction drop per the northstar's §1, not decided
	// here; this is the honest placeholder every dungeon minion uses today.
	KindDungeonMinion = "dungeon-minion"

	DungeonMinionHP     = 40
	DungeonMinionDamage = 8

	// DungeonEliteHPMul/DungeonBossHPMul scale a named hero spawn up from a generic minion --
	// real, deliberate, but arbitrary tuning numbers (no live playtesting exists yet to ground
	// these further; a real, separate balance pass is future work, not blocked on this).
	DungeonEliteHPMul     = 2.5
	DungeonBossHPMul      = 6.0
	DungeonMinionsPerRoom = 2 // rooms between entrance and boss room get this many minions each
)

// GenerateDungeonSpawns builds a real, seeded spawn table for one dungeon instance: generic
// minions in every non-boss room, an optional elite hero in some of them, and the named boss in
// the layout's own BossIdx room. dungeonIndex selects which of DungeonRoster's 8 real named
// dungeons this instance is themed as (index wraps via modulo so any int is safe to pass).
// sceneID is stamped onto every Mob as-is -- this package doesn't decide dungeon scene-ID
// allocation, matching Registry's own existing "caller assigns scene IDs" convention.
func GenerateDungeonSpawns(layout worldapi.DungeonLayout, dungeonIndex int, sceneID int, seed int64) []Mob {
	if len(layout.Rooms) == 0 {
		return nil
	}
	assignment := DungeonRoster[((dungeonIndex%len(DungeonRoster))+len(DungeonRoster))%len(DungeonRoster)]
	rng := rand.New(rand.NewSource(seed))

	var out []Mob
	for i, room := range layout.Rooms {
		pos := Pos{X: float64(room.CenterX), Y: 64, Z: float64(room.CenterZ)}
		if i == layout.BossIdx {
			out = append(out, newDungeonMob(idFor("boss", i), assignment.Boss, pos, DungeonBossHPMul))
			continue
		}
		if i == layout.EntranceIdx {
			// Real, deliberate design choice: the entrance room stays clear, matching every
			// existing zone's own convention of a safe landing spot (Meadow/Caves/Swampville
			// all place their first spawns away from origin -- see each one's own doc comment).
			continue
		}
		for m := 0; m < DungeonMinionsPerRoom; m++ {
			out = append(out, newDungeonMob(idFor("minion", i, m), KindDungeonMinion, jitter(pos, rng), 1.0))
		}
		if len(assignment.Elite) > 0 && rng.Intn(2) == 0 {
			elite := assignment.Elite[rng.Intn(len(assignment.Elite))]
			out = append(out, newDungeonMob(idFor("elite", i), elite, pos, DungeonEliteHPMul))
		}
	}
	for i := range out {
		out[i].SceneID = sceneID
	}
	return out
}

func newDungeonMob(id, kind string, pos Pos, hpMul float64) Mob {
	hp := int(float64(DungeonMinionHP) * hpMul)
	dmg := int(float64(DungeonMinionDamage) * hpMul)
	return Mob{
		ID:          id,
		Kind:        kind,
		Pos:         pos,
		HomePos:     pos,
		HP:          hp,
		MaxHP:       hp,
		State:       StateIdle,
		MeleeDamage: dmg,
	}
}

// jitter nudges a spawn a few units off room center so multiple minions in the same room don't
// stack exactly on top of each other -- real, minimal placement variety, deterministic per-seed
// via the caller's own rng (not a fresh unseeded rand call).
func jitter(p Pos, rng *rand.Rand) Pos {
	p.X += float64(rng.Intn(5) - 2)
	p.Z += float64(rng.Intn(5) - 2)
	return p
}

func idFor(prefix string, nums ...int) string {
	id := prefix
	for _, n := range nums {
		id += "-" + itoa(n)
	}
	return id
}
