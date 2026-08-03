package worldapi

import (
	"strings"
)

// blockNameToID maps Dragonfly/Bedrock block name strings to SHANKPIT VoxelBlock IDs.
// Mirrors packages2/common/block_map.go in SHANKPIT (both tables must stay in sync).
var blockNameToID = map[string]uint16{
	"minecraft:stone":              1,
	"minecraft:cobblestone":        1,
	"minecraft:stone_bricks":       1,
	"minecraft:mossy_cobblestone":  1,
	"minecraft:grass":              2,
	"minecraft:grass_block":        2,
	"minecraft:dirt":               3,
	"minecraft:podzol":             3,
	"minecraft:coarse_dirt":        3,
	"minecraft:oak_log":            17,
	"minecraft:birch_log":          17,
	"minecraft:spruce_log":         17,
	"minecraft:jungle_log":         17,
	"minecraft:acacia_log":         17,
	"minecraft:dark_oak_log":       17,
	"minecraft:log":                17,
	"minecraft:oak_leaves":         18,
	"minecraft:birch_leaves":       18,
	"minecraft:spruce_leaves":      18,
	"minecraft:jungle_leaves":      18,
	"minecraft:acacia_leaves":      18,
	"minecraft:dark_oak_leaves":    18,
	"minecraft:leaves":             18,
	"minecraft:leaves2":            18,
	// Flower ground cover (2026-08-03, founder: "add some block backed flowers to the meadow").
	// New ID 19 -- SHANKPIT's own packages2/common/block_map.go (this table's real sync partner,
	// see this var's own doc comment) has no flower content anywhere in its own scenes, so this ID
	// is GoblinFoxDragon-only for now, an honest gap rather than a guessed cross-repo sync. Poppy
	// and dandelion share one voxel ID (a real VoxelBlock consumer only needs "flower here," not
	// which color) -- the color distinction lives in WorldBlock.BlockName for anything that reads
	// the raw name, and in the client's own render logic (apps2/battlegrounds_gui's
	// town_meadow_flower_positions, driven by array index, not this ID).
	"minecraft:poppy":     19,
	"minecraft:dandelion": 19,
}

// nameToBlockID converts a Dragonfly block name to a SHANKPIT block ID.
// Handles "minecraft:variant_wood" by matching prefix when exact match fails.
// Returns 0 (air) for unknown blocks.
func nameToBlockID(name string) uint16 {
	if id, ok := blockNameToID[name]; ok {
		return id
	}
	// Fallback: check suffix patterns for wood variants ("_log", "_leaves")
	if strings.HasSuffix(name, "_log") {
		return 17
	}
	if strings.HasSuffix(name, "_leaves") {
		return 18
	}
	return 0
}

// WorldBlock is a Dragonfly-side block descriptor passed to DragonflyChunkGenerator.
type WorldBlock struct {
	X, Y, Z   int
	BlockName string // e.g. "minecraft:stone"
}

// DragonflyChunkGenerator implements ChunkGenerator by reading from GFD's world engine.
// Call NewDragonflyChunkGenerator with a WorldStore to wire up real world data.
// When WorldStore is nil it falls back to the built-in procedural generator so the
// /chunks endpoint always returns useful data during development.
type DragonflyChunkGenerator struct {
	// WorldStore is an optional callback that returns all non-air blocks in a chunk.
	// When nil, procedural generation is used (matching SHANKPIT StaticBackend terrain).
	WorldStore func(sceneID, chunkX, chunkZ int) []WorldBlock
}

// NewDragonflyChunkGenerator creates a generator backed by the provided store function.
// Pass nil to use procedural generation.
func NewDragonflyChunkGenerator(store func(sceneID, chunkX, chunkZ int) []WorldBlock) *DragonflyChunkGenerator {
	return &DragonflyChunkGenerator{WorldStore: store}
}

// ChunkBlocks implements ChunkGenerator.
func (d *DragonflyChunkGenerator) ChunkBlocks(sceneID, chunkX, chunkZ int) []VoxelBlock {
	if d.WorldStore != nil {
		raw := d.WorldStore(sceneID, chunkX, chunkZ)
		if raw == nil {
			return nil
		}
		out := make([]VoxelBlock, 0, len(raw))
		for _, b := range raw {
			id := nameToBlockID(b.BlockName)
			if id == 0 {
				continue
			}
			out = append(out, VoxelBlock{
				X: uint8(b.X & 0xF), Y: uint8(b.Y & 0xFF), Z: uint8(b.Z & 0xF),
				BlockID: id,
			})
		}
		return out
	}
	return proceduralChunk(chunkX, chunkZ)
}

// proceduralChunk generates the same flat terrain as SHANKPIT's StaticBackend:
// stone at y=0,1; grass at y=4; oak trees at known positions.
func proceduralChunk(chunkX, chunkZ int) []VoxelBlock {
	const (
		chunkSize   = 16
		groundY     = 4
		stoneID     = 1
		grassID     = 2
		logID       = 17
		leafID      = 18
	)
	blocks := make([]VoxelBlock, 0, 256)
	for z := 0; z < chunkSize; z++ {
		for x := 0; x < chunkSize; x++ {
			blocks = append(blocks,
				VoxelBlock{X: uint8(x), Y: 0, Z: uint8(z), BlockID: stoneID},
				VoxelBlock{X: uint8(x), Y: 1, Z: uint8(z), BlockID: stoneID},
				VoxelBlock{X: uint8(x), Y: groundY, Z: uint8(z), BlockID: grassID},
			)
		}
	}
	for _, t := range proceduralTrees(chunkX, chunkZ) {
		blocks = append(blocks, placeTree(t[0], t[1], groundY+1, logID, leafID)...)
	}
	return blocks
}

func proceduralTrees(chunkX, chunkZ int) [][2]uint8 {
	switch {
	case chunkX == 0 && chunkZ == 0:
		return [][2]uint8{{8, 8}, {3, 12}}
	case chunkX == 1 && chunkZ == 0:
		return [][2]uint8{{6, 5}}
	case chunkX == -1 && chunkZ == -1:
		return [][2]uint8{{11, 4}}
	default:
		return nil
	}
}

func placeTree(tx, tz uint8, baseY int, logID, leafID uint16) []VoxelBlock {
	out := make([]VoxelBlock, 0, 12)
	for y := 0; y < 4; y++ {
		out = append(out, VoxelBlock{X: tx, Y: uint8(baseY + y), Z: tz, BlockID: logID})
	}
	leafY := uint8(baseY + 4)
	for dz := -1; dz <= 1; dz++ {
		for dx := -1; dx <= 1; dx++ {
			lx := int(tx) + dx
			lz := int(tz) + dz
			if lx < 0 || lx > 15 || lz < 0 || lz > 15 {
				continue
			}
			out = append(out, VoxelBlock{X: uint8(lx), Y: leafY, Z: uint8(lz), BlockID: leafID})
		}
	}
	return out
}
