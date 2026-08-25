package mob

import (
	"encoding/json"
	"os"
	"time"
)

const (
	crystalGoblinSwingDelay = 3 * time.Second
	crystalFoxSwingDelay    = 2 * time.Second
)

// crystalNPC/crystalSeed mirror apps2/crystal/main.go's own CrystalSeed export
// format exactly (S189-07) -- kept as a private, minimal decode-only shape here
// rather than importing crystal as a package, since crystal is a standalone
// `package main` binary, not a library this module can import; the JSON
// contract between the two is the real interface, matching the deliberate
// crystal->MUD split BACKLOG.md's own scoping settled on ("the crystal output
// can be used to provide a map to the mud" -- founder). Terrain grids
// (City/Entropy/Rubble) are decoded but not consumed yet -- MeadowCrystalSpawns
// only needs the NPC list; a future terrain-flavor pass can read them from the
// same seed file without a format change.
type crystalNPC struct {
	X, Y int
	Kind string
}

type crystalSeed struct {
	Tick    int
	Width   int
	Height  int
	Goblins []crystalNPC
	Foxes   []crystalNPC
}

// crystalGridScale/crystalGridOffset map crystal's 44x44 grid (0..43 in each
// axis) onto the same real-world coordinate space MeadowWormSpawns' own ring
// already uses (~35-unit radius around the town centre at (0,2,0)) -- centers
// the grid at its own midpoint (21.5,21.5) and scales the 44-wide grid to a
// 70-unit span (-35..+35), matching the worm ring's own real radius rather
// than inventing an unrelated scale.
const (
	crystalGridOffset = 21.5
	crystalGridScale  = 70.0 / 44.0
)

func crystalWorldPos(gridX, gridY int) Pos {
	return Pos{
		X: (float64(gridX) - crystalGridOffset) * crystalGridScale,
		Y: 2, // ground level, matching NewWorm's own Y=2 convention
		Z: (float64(gridY) - crystalGridOffset) * crystalGridScale,
	}
}

// Goblin base stats, distinct per kind so crystal's own Scavenger/Tinkerer/
// Raider/Merchant typing carries real gameplay weight rather than being
// cosmetic-only. Raider is the aggressive fighter (higher HP/damage, real
// aggro range); the other three are low-threat, wandering-flavor NPCs
// (near-zero aggro range) -- crystal's own simulation already models them as
// non-combat roles (scavenging rubble, fixing power, trading), so making them
// all fight players the same as a worm would contradict that.
type crystalGoblinStats struct {
	hp, damage        int
	moveSpeed         float64
	aggroRange        float64
	leashRange        float64
}

var goblinStatsByKind = map[string]crystalGoblinStats{
	"scavenger": {hp: 60, damage: 4, moveSpeed: 1.2, aggroRange: 0, leashRange: 15},
	"tinkerer":  {hp: 55, damage: 3, moveSpeed: 1.0, aggroRange: 0, leashRange: 15},
	"raider":    {hp: 110, damage: 12, moveSpeed: 1.8, aggroRange: 10, leashRange: 22},
	"merchant":  {hp: 50, damage: 2, moveSpeed: 0.8, aggroRange: 0, leashRange: 12},
}

// Foxes are all non-aggressive wildlife flavor (crystal's own Fox type has no
// combat-relevant fields -- Energy only) -- low HP, no aggro range, present
// for atmosphere and to make the seeded zone visibly alive, not as a fight.
const (
	foxHP         = 30
	foxMoveSpeed  = 2.2
)

func newCrystalGoblin(id string, npc crystalNPC) Mob {
	stats, ok := goblinStatsByKind[npc.Kind]
	if !ok {
		stats = goblinStatsByKind["scavenger"]
	}
	pos := crystalWorldPos(npc.X, npc.Y)
	return Mob{
		ID:          id,
		Kind:        "goblin-" + npc.Kind,
		SceneID:     0, // Meadow
		Pos:         pos,
		HomePos:     pos,
		HP:          stats.hp,
		MaxHP:       stats.hp,
		State:       StateIdle,
		AggroRange:  stats.aggroRange,
		LeashRange:  stats.leashRange,
		MeleeRange:  2.0,
		MoveSpeed:   stats.moveSpeed,
		SwingDelay:  crystalGoblinSwingDelay,
		MeleeDamage: stats.damage,
	}
}

func newCrystalFox(id string, npc crystalNPC) Mob {
	pos := crystalWorldPos(npc.X, npc.Y)
	return Mob{
		ID:          id,
		Kind:        "fox-" + npc.Kind,
		SceneID:     0, // Meadow
		Pos:         pos,
		HomePos:     pos,
		HP:          foxHP,
		MaxHP:       foxHP,
		State:       StateIdle,
		AggroRange:  0, // wildlife flavor, non-aggressive
		LeashRange:  15,
		MeleeRange:  1.5,
		MoveSpeed:   foxMoveSpeed,
		SwingDelay:  crystalFoxSwingDelay,
		MeleeDamage: 1,
	}
}

// MeadowCrystalSpawns loads a crystal-generated world seed (see
// apps2/crystal/main.go's own `-seed`/`-seed-out` flags) and returns the
// goblin+fox NPCs it describes as real Meadow (zone 0) mobs, positioned by
// mapping crystal's 44x44 grid onto the same real-world coordinate space
// MeadowWormSpawns' own perimeter ring uses. Returns (nil, nil) if seedPath
// is empty or the file doesn't exist -- seeding the Meadow this way is
// optional, additive content, not a hard dependency world init should fail
// without (matches the founder's own "quickly generate" framing, not a
// mandatory always-on system).
func MeadowCrystalSpawns(seedPath string) ([]Mob, error) {
	if seedPath == "" {
		return nil, nil
	}
	f, err := os.Open(seedPath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var seed crystalSeed
	if err := json.NewDecoder(f).Decode(&seed); err != nil {
		return nil, err
	}

	mobs := make([]Mob, 0, len(seed.Goblins)+len(seed.Foxes))
	for i, g := range seed.Goblins {
		mobs = append(mobs, newCrystalGoblin(crystalGoblinID(i), g))
	}
	for i, fx := range seed.Foxes {
		mobs = append(mobs, newCrystalFox(crystalFoxID(i), fx))
	}
	return mobs, nil
}

func crystalGoblinID(i int) string { return "crystal-goblin-meadow-" + itoa(i) }
func crystalFoxID(i int) string    { return "crystal-fox-meadow-" + itoa(i) }
