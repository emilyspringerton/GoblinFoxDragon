// Package telecrystal implements the server-side telecrystal travel registry.
//
// Crystal data is sourced from GoblinFoxDragon/docs2/specs/TELECRYSTAL_NETWORK_SPEC.md.
// Scene IDs match the zone package: 0=Meadow(Town), 1=Hills, 2=Caves(Mines), 3=Swamp(Docks).
package telecrystal

import (
	"errors"
	"math"
)

var (
	ErrUnknownCrystal   = errors.New("telecrystal: unknown crystal ID")
	ErrWrongScene       = errors.New("telecrystal: player not in crystal source scene")
	ErrInsufficientGold = errors.New("telecrystal: insufficient gold for cast cost")
)

// Vec3 is a 3D world position (X, Y, Z in scene-local coordinates).
type Vec3 struct{ X, Y, Z float64 }

// Crystal is one entry in the telecrystal network.
type Crystal struct {
	ID          string
	SourceScene int
	Position    Vec3    // center of interaction ring (scene-local)
	Radius      float64 // interaction radius in world units
	Prompt      string  // HUD prompt shown to player
	CastCost    int     // gold deducted on successful travel
	TargetScene int
	SpawnPos    Vec3    // where the player appears in TargetScene
	SpawnYaw    float64 // degrees
	SpawnPitch  float64
	TargetName  string
}

// Dist2D returns horizontal (XZ) distance from c.Position to pos.
func (c Crystal) Dist2D(pos Vec3) float64 {
	dx := c.Position.X - pos.X
	dz := c.Position.Z - pos.Z
	return math.Sqrt(dx*dx + dz*dz)
}

// InRange returns true if pos is within c.Radius horizontally of c.Position.
func (c Crystal) InRange(pos Vec3) bool {
	return c.Dist2D(pos) <= c.Radius
}

// Scene IDs matching server/zone.
const (
	SceneMeadow      = 0 // Town / SCENE_CITY (the OLDER "Town" -- apps/lobby's SHANKPIT-style scene)
	SceneHills       = 1
	SceneCaves       = 2 // Mines / SCENE_MINES
	SceneSwamp       = 3 // Docks / SCENE_DOCKS
	SceneNewHandington = 4 // New Handington -- apps2/battlegrounds_gui's own, newer Town
)

// Registry is the global telecrystal network (from TELECRYSTAL_NETWORK_SPEC.md).
var Registry = []Crystal{
	{
		ID:          "TELECRYSTAL_ID_TOWN_TO_MINES",
		SourceScene: SceneMeadow, Position: Vec3{270, 0, 60}, Radius: 14,
		Prompt: "G: TELEPORT MINES", CastCost: 1000,
		TargetScene: SceneCaves, SpawnPos: Vec3{0, 0, -92},
		SpawnYaw: 0, SpawnPitch: 0, TargetName: "MINES",
	},
	{
		ID:          "TELECRYSTAL_ID_MINES_RETURN_TOWN",
		SourceScene: SceneCaves, Position: Vec3{0, 0, -92}, Radius: 10,
		Prompt: "G: RETURN TOWN", CastCost: 900,
		TargetScene: SceneMeadow, SpawnPos: Vec3{270, 0, 60},
		SpawnYaw: 180, SpawnPitch: 0, TargetName: "TOWN",
	},
	{
		ID:          "TELECRYSTAL_ID_MINES_STUB_CRYSTAL",
		SourceScene: SceneCaves, Position: Vec3{0, 0, 178}, Radius: 10,
		Prompt: "G: CRYSTAL SHUNT", CastCost: 900,
		TargetScene: SceneCaves, SpawnPos: Vec3{0, 0, 44},
		SpawnYaw: 180, SpawnPitch: 0, TargetName: "MINE SPLIT",
	},
	{
		ID:          "TELECRYSTAL_ID_MINES_SPLIT_CRYSTAL",
		SourceScene: SceneCaves, Position: Vec3{0, 0, 44}, Radius: 8,
		Prompt: "G: RETURN ENTRY", CastCost: 900,
		TargetScene: SceneCaves, SpawnPos: Vec3{0, 0, -92},
		SpawnYaw: 180, SpawnPitch: 0, TargetName: "MINE ENTRY",
	},
	{
		ID:          "TELECRYSTAL_ID_TOWN_TO_DOCKS",
		SourceScene: SceneMeadow, Position: Vec3{308, 0, 246}, Radius: 14,
		Prompt: "G: TELEPORT DOCKS", CastCost: 1000,
		TargetScene: SceneSwamp, SpawnPos: Vec3{-178, 0, -168},
		SpawnYaw: 95, SpawnPitch: 0, TargetName: "DOCKS",
	},
	{
		ID:          "TELECRYSTAL_ID_DOCKS_RETURN_TOWN",
		SourceScene: SceneSwamp, Position: Vec3{-178, 0, -168}, Radius: 11,
		Prompt: "G: RETURN TOWN", CastCost: 900,
		TargetScene: SceneMeadow, SpawnPos: Vec3{300, 0, 238},
		SpawnYaw: 210, SpawnPitch: 0, TargetName: "TOWN",
	},
	// New Handington <-> Meadow (2026-08-02, founder: "how do we get from town to the starter
	// zone? have one of the gates act as a telecrystal"): New Handington (zone 4,
	// apps2/battlegrounds_gui's own Town) predates this network entirely -- it has no telecrystal
	// of its own yet, and Meadow (zone 0) is the REAL starter zone real telnet players spawn into
	// (MeadowWormSpawns' worm-meadow-0..7), not the same thing as New Handington's own decorative
	// Worm Hut. Position matches New Handington's real "Dragon Gate" building
	// (TOWN_BUILDINGS, apps2/battlegrounds_gui/src/main.c) -- one of Town's own gate buildings,
	// per founder direction, made real instead of purely decorative. Free (CastCost 0): unlike
	// the older network's Flow-gated crystals, there's no real economy reason to charge for a
	// starter-zone shuttle a level-1 character needs before they'd ever have Flow to spend.
	{
		ID:          "TELECRYSTAL_ID_HANDINGTON_TO_MEADOW",
		SourceScene: SceneNewHandington, Position: Vec3{-40, 0, -50}, Radius: 12,
		Prompt: "G: TELEPORT MEADOW (STARTER ZONE)", CastCost: 0,
		TargetScene: SceneMeadow, SpawnPos: Vec3{0, 2, 0},
		SpawnYaw: 0, SpawnPitch: 0, TargetName: "MEADOW",
	},
	{
		ID:          "TELECRYSTAL_ID_MEADOW_RETURN_HANDINGTON",
		SourceScene: SceneMeadow, Position: Vec3{0, 2, 0}, Radius: 12,
		Prompt: "G: RETURN NEW HANDINGTON", CastCost: 0,
		TargetScene: SceneNewHandington, SpawnPos: Vec3{-40, 0, -50},
		SpawnYaw: 180, SpawnPitch: 0, TargetName: "NEW HANDINGTON",
	},
}

// Lookup returns the Crystal with the given ID, or ErrUnknownCrystal.
func Lookup(id string) (Crystal, error) {
	for _, c := range Registry {
		if c.ID == id {
			return c, nil
		}
	}
	return Crystal{}, ErrUnknownCrystal
}

// InScene returns all crystals whose SourceScene matches sceneID.
func InScene(sceneID int) []Crystal {
	var out []Crystal
	for _, c := range Registry {
		if c.SourceScene == sceneID {
			out = append(out, c)
		}
	}
	return out
}

// Validate checks that a player in sceneID at pos has enough gold for crystal.
// Returns the crystal or an error. Does NOT deduct gold.
func Validate(crystalID string, sceneID int, playerPos Vec3, goldBalance int) (Crystal, error) {
	c, err := Lookup(crystalID)
	if err != nil {
		return Crystal{}, err
	}
	if c.SourceScene != sceneID {
		return Crystal{}, ErrWrongScene
	}
	if goldBalance < c.CastCost {
		return Crystal{}, ErrInsufficientGold
	}
	return c, nil
}
