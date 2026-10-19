package main

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"

	"dragonsnshit/server/cartography"
	"dragonsnshit/server/homepoint"
	"dragonsnshit/server/mob"
)

// TestOwlZonesBootAndWalk boots the real initWorld (from the repo root, so data/*.json load) and
// walks a player Meadow -> Fledgling Field, checking the entry text carries the zone's smell.
func TestOwlZonesBootAndWalk(t *testing.T) {
	wd, _ := os.Getwd()
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(wd) })

	oldGW := gw
	gw = initWorld()
	t.Cleanup(func() { gw = oldGW })

	for _, zid := range []int{10, 11, 12, 13, 14, 15, 16, 17, 18} {
		if _, ok := gw.zoneMgr.Get(zid); !ok {
			t.Fatalf("zone %d not registered", zid)
		}
		if len(gw.nmSpawns[zid]) != 2 {
			t.Errorf("zone %d: want 2 NM spawns registered, got %d", zid, len(gw.nmSpawns[zid]))
		}
		if gw.mobRegs[zid] == nil {
			t.Fatalf("zone %d has no mob registry", zid)
		}
	}

	p := newTestPlayer()
	p.slot = "walker"
	p.name = "Walker"
	p.zoneID = 0
	p.homePoint = homepoint.NewState(0)
	p.atlas = cartography.NewAtlas()
	p.combat = &mob.PlayerCombat{}
	var buf bytes.Buffer
	p.w = bufio.NewWriter(&buf)
	gw.players = map[string]*player{p.slot: p}
	_ = gw.zoneMgr.Transfer(p.slot, 0)

	cmdGo(p, "west")
	p.w.Flush()
	out := buf.String()
	if p.zoneID != 10 {
		t.Fatalf("walking west from the Meadow should reach zone 10, at %d\n%s", p.zoneID, out)
	}
	for _, want := range []string{"Fledgling Field", "You smell:", "barley", "Recommended levels 3-8"} {
		if !strings.Contains(out, want) {
			t.Errorf("entry text missing %q:\n%s", want, out)
		}
	}
}
