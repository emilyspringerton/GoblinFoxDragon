package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"dragonsnshit/server/itemdef"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/mobdrop"
	"dragonsnshit/server/spawn"
)

func loadOwlItems(t *testing.T) *itemdef.Registry {
	t.Helper()
	r := itemdef.NewRegistry()
	if err := r.LoadFile("../../data/items.json"); err != nil {
		t.Fatalf("load items.json: %v", err)
	}
	return r
}

func TestOwlZoneExitsAreBidirectional(t *testing.T) {
	opposite := map[string]string{"north": "south", "south": "north", "east": "west", "west": "east", "up": "down", "down": "up"}
	for _, zid := range mob.OwlZoneIDs() {
		for dir, dest := range exits[zid] {
			back, ok := exits[dest][opposite[dir]]
			if !ok || back != zid {
				t.Errorf("zone %d %s -> %d has no matching %s back (got %d)", zid, dir, dest, opposite[dir], back)
			}
		}
	}
}

func TestOwlZonesHaveDescSmellNPCAndVendor(t *testing.T) {
	for _, zid := range mob.OwlZoneIDs() {
		if zoneDesc[zid] == "" || zoneSmell[zid] == "" || owlZoneNames[zid] == "" {
			t.Errorf("zone %d missing desc/smell/name", zid)
		}
		if zoneVendorNPC(zid) == nil {
			t.Errorf("zone %d has no vendor NPC", zid)
		}
	}
}

func TestOwlShopItemsResolveAndExoticsAreNMOnly(t *testing.T) {
	reg := loadOwlItems(t)
	vendored := map[string]bool{}
	for _, zid := range mob.OwlZoneIDs() {
		npc := zoneVendorNPC(zid)
		for _, vi := range npcVendorCatalog[npc.ID] {
			vendored[vi.ID] = true
			if _, ok := reg.ByName(vi.ID); !ok && itemName(vi.ID) == vi.ID {
				t.Errorf("zone %d vendor sells %q which resolves nowhere", zid, vi.ID)
			}
		}
	}
	for _, d := range reg.All() {
		if d.ID >= 1300 && d.ID < 1400 {
			if vendored[strings.ReplaceAll(strings.ToLower(d.Name), " ", "-")] {
				t.Errorf("exotic %q must be NM-only but a shop sells it", d.Name)
			}
			hasRare := false
			for _, f := range d.FlagsRaw {
				hasRare = hasRare || f == "rare"
			}
			if !hasRare {
				t.Errorf("exotic %q should be flagged rare", d.Name)
			}
		}
	}
}

func TestOwlNMsWiredEndToEnd(t *testing.T) {
	reg := loadOwlItems(t)
	var rules []spawn.Rule
	raw, err := os.ReadFile("../../data/mob_spawns.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &rules); err != nil {
		t.Fatal(err)
	}
	nmRules := map[string]bool{}
	for _, r := range rules {
		if r.NM != nil {
			nmRules[r.NM.ID] = true
		}
	}
	drops := mobdrop.NewRegistry()
	if err := drops.LoadFile("../../data/mob_drops.json"); err != nil {
		t.Fatal(err)
	}
	if len(mob.OwlNMDefs) != 18 {
		t.Fatalf("want 18 owl NMs, got %d", len(mob.OwlNMDefs))
	}
	for _, d := range mob.OwlNMDefs {
		if _, ok := mob.OwlNMTemplate(d.ID); !ok {
			t.Errorf("%s: no template", d.ID)
		}
		if !nmRules[d.ID] {
			t.Errorf("%s: no spawn rule in data/mob_spawns.json", d.ID)
		}
		got := drops.DropsFor(d.Kind)
		if len(got) < 3 {
			t.Errorf("%s: want >=2 exotics + token, got %d drops", d.Kind, len(got))
		}
		for _, it := range got {
			if it.ID == "nm-owl-token" {
				continue
			}
			if _, ok := reg.ByName(it.ID); !ok {
				t.Errorf("%s drops %q which is not in items.json", d.Kind, it.ID)
			}
		}
	}
}

func TestOwlQuestsReferenceRealThings(t *testing.T) {
	reg := loadOwlItems(t)
	nmKinds := map[string]bool{}
	for _, d := range mob.OwlNMDefs {
		nmKinds[d.Kind] = true
	}
	for _, q := range questBank.All() {
		if !strings.HasPrefix(q.ID, "owl-") {
			continue
		}
		if npcByID(q.GiverNPCID) == nil {
			t.Errorf("%s: unknown giver %s", q.ID, q.GiverNPCID)
		}
		if _, ok := reg.ByName(q.RewardItem); !ok {
			t.Errorf("%s: reward %q not in items.json", q.ID, q.RewardItem)
		}
		for k := range q.RequireKills {
			if !nmKinds[k] {
				t.Errorf("%s: kill target %q is not an owl NM kind", q.ID, k)
			}
		}
	}
}
