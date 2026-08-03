// Package worldapi exposes the GoblinFoxDragon world state over HTTP for the
// SHANKPIT DragonflyBackend. Endpoint: GET /chunks?scene=N&cx=X&cz=Z
//
// Response: JSON array of VoxelBlock {X, Y, Z uint8; BlockID uint16}
// Returns 200 with block array (may be empty), or 204 when no world data is
// loaded for that scene.
//
// Start with:
//
//	srv := worldapi.New(generator)
//	http.ListenAndServe(":7070", srv)
package worldapi

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// VoxelBlock matches SHANKPIT's system.VoxelBlock wire format.
type VoxelBlock struct {
	X, Y, Z uint8
	BlockID  uint16
}

// ChunkGenerator provides chunk data for a given scene + chunk coordinate.
// Return nil to signal "no data" (SHANKPIT falls back to procedural generator).
type ChunkGenerator interface {
	ChunkBlocks(sceneID, chunkX, chunkZ int) []VoxelBlock
}

// Server is an http.Handler serving the world chunk API.
type Server struct {
	gen ChunkGenerator
}

func New(gen ChunkGenerator) *Server {
	return &Server{gen: gen}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/chunks":
		s.serveChunks(w, r)
	case "/heightmap":
		s.serveHeightmap(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) serveChunks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	scene, _ := strconv.Atoi(q.Get("scene"))
	cx, _ := strconv.Atoi(q.Get("cx"))
	cz, _ := strconv.Atoi(q.Get("cz"))

	blocks := s.gen.ChunkBlocks(scene, cx, cz)
	if blocks == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
}

// heightmapResponse is the /heightmap wire format (SMOOTH_TERRAIN_NORTHSTAR.md §3.1). Biome is
// a single value for the whole chunk, not per-column -- ProceduralWorldStore has no per-column
// biome mixing today (sceneID is the closest thing to a biome selector, see the northstar's own
// §1), so a 256-entry array of one repeated value would just be wasted bytes on the wire.
type heightmapResponse struct {
	Height [256]uint8 `json:"height"`
	Biome  int        `json:"biome"`
}

// serveHeightmap calls HeightmapChunk directly against ProceduralWorldStore's own column-derived
// scenes, bypassing the ChunkGenerator interface entirely -- unlike /chunks, this isn't a generic
// abstraction over "some world store," it's specifically exposing the height math
// ProceduralWorldStore's scenes already compute internally (see heightmap.go's own doc comment).
func (s *Server) serveHeightmap(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	scene, _ := strconv.Atoi(q.Get("scene"))
	cx, _ := strconv.Atoi(q.Get("cx"))
	cz, _ := strconv.Atoi(q.Get("cz"))

	heights, ok := HeightmapChunk(scene, cx, cz)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(heightmapResponse{Height: heights, Biome: scene})
}
