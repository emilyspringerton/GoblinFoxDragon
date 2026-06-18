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
	if r.URL.Path != "/chunks" {
		http.NotFound(w, r)
		return
	}
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
