package worldapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dragonsnshit/server/worldapi"
)

type fakeGen struct {
	blocks []worldapi.VoxelBlock
}

func (f *fakeGen) ChunkBlocks(_, _, _ int) []worldapi.VoxelBlock { return f.blocks }

type nilGen struct{}

func (n *nilGen) ChunkBlocks(_, _, _ int) []worldapi.VoxelBlock { return nil }

func TestChunksEndpoint_returnsBlocks(t *testing.T) {
	want := []worldapi.VoxelBlock{{X: 1, Y: 2, Z: 3, BlockID: 7}}
	srv := worldapi.New(&fakeGen{blocks: want})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/chunks?scene=0&cx=0&cz=0", nil)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got []worldapi.VoxelBlock
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("unexpected blocks: %+v", got)
	}
}

func TestChunksEndpoint_nilReturns204(t *testing.T) {
	srv := worldapi.New(&nilGen{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/chunks?scene=0&cx=0&cz=0", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestHeightmapEndpoint_returnsHeights(t *testing.T) {
	srv := worldapi.New(&nilGen{}) // heightmap bypasses ChunkGenerator entirely, see worldapi.go
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/heightmap?scene=0&cx=0&cz=0", nil)
	srv.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var got struct {
		Height [256]uint8 `json:"height"`
		Biome  int        `json:"biome"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Biome != 0 {
		t.Fatalf("expected biome 0, got %d", got.Biome)
	}
	if got.Height[0] != 4 {
		t.Fatalf("expected meadow height 4, got %d", got.Height[0])
	}
}

func TestHeightmapEndpoint_caves_returns204(t *testing.T) {
	srv := worldapi.New(&nilGen{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/heightmap?scene=2&cx=0&cz=0", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
}

func TestUnknownPath_returns404(t *testing.T) {
	srv := worldapi.New(&nilGen{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/other", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}
