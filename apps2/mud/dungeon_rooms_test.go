package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"dragonsnshit/server/mob"
	"dragonsnshit/server/mobdrop"
)

func TestDungeonsAreSixConnectedRoomsEach(t *testing.T) {
	opposite := map[string]string{"north": "south", "south": "north", "east": "west", "west": "east"}
	for i := range mob.DungeonRoster {
		rooms := map[int]bool{}
		for _, r := range mob.DungeonRoomTable {
			if r.Dungeon == i {
				rooms[r.ZoneID] = true
				if zoneDesc[r.ZoneID] == "" || zoneSmell[r.ZoneID] == "" {
					t.Errorf("dungeon %d room %d (%s) needs a description and a smell", i, r.Index, r.Name)
				}
			}
		}
		if len(rooms) != 6 {
			t.Errorf("dungeon %d has %d rooms, want 6", i, len(rooms))
		}
		// every exit leads somewhere that leads back (the entrance's "out" is one-way to town)
		for z := range rooms {
			for dir, dest := range exits[z] {
				if dir == "out" {
					continue
				}
				back, ok := exits[dest][opposite[dir]]
				if !ok || back != z {
					t.Errorf("room %d %s -> %d has no way back", z, dir, dest)
				}
			}
		}
	}
	if _, ok := exits[dungeonZoneBase]["out"]; !ok {
		t.Errorf("the entrance needs a way out")
	}
}

func TestDungeonDifficultyClimbsRoomByRoomAndDungeonByDungeon(t *testing.T) {
	prevTop := 0
	for i := range mob.DungeonRoster {
		last := 0
		for _, r := range mob.DungeonRoomTable {
			if r.Dungeon != i || r.Index == 0 {
				continue
			}
			if r.Level < last {
				t.Errorf("dungeon %d room %d (L%d) is easier than the room before it (L%d)", i, r.Index, r.Level, last)
			}
			last = r.Level
		}
		first := mob.DungeonLevelBands[i][0]
		if i > 0 && first <= prevTop-12 {
			t.Errorf("dungeon %d should not start far below the previous dungeon's top", i)
		}
		prevTop = mob.DungeonLevelBands[i][1]
	}
	for i := 1; i < len(mob.DungeonLevelBands); i++ {
		if mob.DungeonLevelBands[i][0] <= mob.DungeonLevelBands[i-1][0] {
			t.Errorf("dungeon bands must rise")
		}
	}
}

func TestDungeonSpawnsUseRosterHeroesWithCharacterNames(t *testing.T) {
	mobs := mob.DungeonRoomSpawns(300 + 1*10 + 5) // Frequency Table boss hall
	var boss *mob.Mob
	for i := range mobs {
		if strings.Contains(mobs[i].Kind, "Paimon") {
			boss = &mobs[i]
		}
	}
	if boss == nil {
		t.Fatalf("Frequency Table boss hall should hold Paimon, got %v", kinds(mobs))
	}
	if boss.CCResist < 0.5 {
		t.Errorf("a boss resists crowd control")
	}
	elite := mob.DungeonRoomSpawns(300 + 1*10 + 3)
	found := false
	for _, m := range elite {
		if strings.Contains(m.Kind, "Vassago") || strings.Contains(m.Kind, "Beleth") || strings.Contains(m.Kind, "Zagan") || strings.Contains(m.Kind, "Buer") {
			found = true
		}
		if strings.Contains(m.Kind, "ARENA_HERO") {
			t.Errorf("raw hero id leaked into a name: %s", m.Kind)
		}
	}
	if !found {
		t.Errorf("elite chamber should hold a roster elite: %v", kinds(elite))
	}
	// roster edits (IDUNA) change who spawns
	old := mob.DungeonRoster
	defer func() { mob.DungeonRoster = old }()
	edited := append([]mob.DungeonBossAssignment(nil), old...)
	edited[1] = mob.DungeonBossAssignment{Name: edited[1].Name, Boss: "ARENA_HERO_CAIN"}
	mob.DungeonRoster = edited
	for _, m := range mob.DungeonRoomSpawns(300 + 1*10 + 5) {
		if strings.Contains(m.Kind, "Cain") {
			return
		}
	}
	t.Errorf("an edited roster should change the boss")
}

func kinds(ms []mob.Mob) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.Kind)
	}
	return out
}

// Every dungeon monster has a drop table and the bosses drop real weapons from the item database.
func TestDungeonMobsDropWeaponsFromTheItemDatabase(t *testing.T) {
	reg := mobdrop.NewRegistry()
	if err := reg.LoadFile("../../data/mob_drops.json"); err != nil {
		t.Fatal(err)
	}
	weapons := map[string]bool{}
	for _, it := range loadRawItems(t) {
		if it.Cat == "weapon" {
			weapons[slugOf(it.Name)] = true
		}
	}
	for i := range mob.DungeonRoster {
		bossSeen := false
		for _, room := range mob.DungeonRoomTable {
			if room.Dungeon != i {
				continue
			}
			for _, m := range mob.DungeonRoomSpawns(room.ZoneID) {
				drops := reg.DropsFor(m.Kind)
				if len(drops) == 0 {
					t.Errorf("%s (dungeon %d room %d) has no drop table", m.Kind, i, room.Index)
					continue
				}
				n := 0
				for _, d := range drops {
					if weapons[d.ID] {
						n++
					}
				}
				if n == 0 {
					t.Errorf("%s drops no weapon", m.Kind)
				}
				if room.Index == 5 && strings.Contains(strings.ToLower(m.Kind), strings.ToLower(strings.Fields(mob.HeroName(mob.DungeonRoster[i].Boss))[0])) {
					bossSeen = true
					if n < 3 {
						t.Errorf("boss %s should drop a few weapons, got %d", m.Kind, n)
					}
				}
			}
		}
		if !bossSeen {
			t.Errorf("dungeon %d boss not found in its boss hall", i)
		}
	}
	// the top dungeon's boss offers the high-end katanas
	top := reg.DropsFor(mob.HeroName(mob.DungeonRoster[7].Boss))
	has := map[string]bool{}
	for _, d := range top {
		has[d.ID] = true
	}
	if !has["mikazuki"] && !has["juzumaru"] && !has["onimaru"] {
		t.Errorf("the Proving Grounds boss should drop endgame katanas, drops=%v", top)
	}
}

func TestDungeonSpawnRulesAreToggleableInIduna(t *testing.T) {
	b, err := os.ReadFile("../../data/mob_spawns.json")
	if err != nil {
		t.Fatal(err)
	}
	var rules []struct {
		ZoneID int    `json:"zone_id"`
		Kind   string `json:"kind"`
	}
	_ = json.Unmarshal(b, &rules)
	have := map[[2]interface{}]bool{}
	for _, r := range rules {
		have[[2]interface{}{r.ZoneID, r.Kind}] = true
	}
	for _, room := range mob.DungeonRoomTable {
		for _, g := range room.Mobs {
			if g.Role == "minion" && !have[[2]interface{}{room.ZoneID, g.Kind}] {
				t.Errorf("no spawn rule for %s in zone %d (IDUNA could not toggle it)", g.Kind, room.ZoneID)
			}
		}
	}
}
