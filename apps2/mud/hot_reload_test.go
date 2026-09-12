package main

import (
	"os"
	"path/filepath"
	"testing"

	"dragonsnshit/server/itemdef"
	"dragonsnshit/server/mobdrop"
)

// TestReloadableDataFiles_PicksUpChangedItemStats guards the real, founder-requested hot-reload
// feature (S412-05): a real edit to data/items.json (e.g. tuning the starting Sword's own
// {"attack":10,"str":1}) must take effect on the next reload without a server restart. Uses a
// real temp file + os.Chdir so this exercises the exact same relative-path LoadFile call
// reloadableDataFiles/main() use, not a mocked path.
func TestReloadableDataFiles_PicksUpChangedItemStats(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "data"), 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	itemsPath := filepath.Join(dir, "data", "items.json")
	if err := os.WriteFile(itemsPath, []byte(`[{"id":3,"name":"Sword","category":"weapon","stack_size":1,"stats":{"attack":10}}]`), 0o644); err != nil {
		t.Fatalf("write items.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "data", "mob_drops.json"), []byte(`[]`), 0o644); err != nil {
		t.Fatalf("write mob_drops.json: %v", err)
	}

	oldWD, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldWD) })

	oldItemReg, oldDropReg := itemdefReg, mobDropReg
	itemdefReg = itemdef.NewRegistry()
	mobDropReg = mobdrop.NewRegistry()
	t.Cleanup(func() { itemdefReg, mobDropReg = oldItemReg, oldDropReg })

	if err := itemdefReg.LoadFile("data/items.json"); err != nil {
		t.Fatalf("initial load: %v", err)
	}
	before, _ := itemdefReg.ByID(3)
	if before.Stats["attack"] != 10 {
		t.Fatalf("precondition failed: expected attack=10 before edit, got %v", before.Stats)
	}

	// Real, live edit -- exactly what an operator tuning the sword's own stats would do.
	if err := os.WriteFile(itemsPath, []byte(`[{"id":3,"name":"Sword","category":"weapon","stack_size":1,"stats":{"attack":25}}]`), 0o644); err != nil {
		t.Fatalf("rewrite items.json: %v", err)
	}

	reloadableDataFiles()

	after, ok := itemdefReg.ByID(3)
	if !ok {
		t.Fatal("item 3 missing after reload")
	}
	if after.Stats["attack"] != 25 {
		t.Errorf("attack after hot reload = %d, want 25 (the real edited value)", after.Stats["attack"])
	}
}

func TestReloadableDataFiles_MissingFileDoesNotPanicOrClearExistingData(t *testing.T) {
	dir := t.TempDir()
	oldWD, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(oldWD) })

	oldItemReg, oldDropReg := itemdefReg, mobDropReg
	itemdefReg = itemdef.NewRegistry()
	if err := itemdefReg.LoadJSON([]byte(`[{"id":3,"name":"Sword","category":"weapon","stack_size":1,"stats":{"attack":10}}]`)); err != nil {
		t.Fatalf("seed: %v", err)
	}
	mobDropReg = mobdrop.NewRegistry()
	t.Cleanup(func() { itemdefReg, mobDropReg = oldItemReg, oldDropReg })

	reloadableDataFiles() // no data/ dir exists here at all -- must not panic

	still, ok := itemdefReg.ByID(3)
	if !ok || still.Stats["attack"] != 10 {
		t.Error("a failed reload (missing file) should keep the previously loaded data intact")
	}
}
