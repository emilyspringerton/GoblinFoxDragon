package main

import (
	"bufio"
	"bytes"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"dragonsnshit/server/cartography"
	"dragonsnshit/server/chat"
	"dragonsnshit/server/homepoint"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/xp"
	"dragonsnshit/server/zone"
)

func shipWorld(t *testing.T, level, flow int) (*player, *bytes.Buffer) {
	t.Helper()
	oldGW := gw
	gw = &world{zoneMgr: zone.New(zone.DefaultZones()), mobRegs: map[int]*mob.Registry{}, players: map[string]*player{},
		charIDBySlot: map[string]string{}, rng: rand.New(rand.NewSource(1)), chatRouter: chat.New()}
	for _, zid := range mob.OwlZoneIDs() {
		_ = gw.zoneMgr.AddZone(&zone.Zone{ID: zid, Name: owlZoneNames[zid]})
		gw.mobRegs[zid] = mob.New()
	}
	t.Cleanup(func() { gw = oldGW })
	p := newTestPlayer()
	p.slot, p.name, p.zoneID = "sailor", "Sailor", zoneDocks
	p.charXP = &xp.CharXP{Level: level}
	p.flow = flow
	p.homePoint = homepoint.NewState(0)
	p.combat = &mob.PlayerCombat{}
	p.atlas = cartography.NewAtlas()
	var buf bytes.Buffer
	p.w = bufio.NewWriter(&buf)
	gw.players[p.slot] = p
	_ = gw.zoneMgr.Transfer(p.slot, zoneDocks)
	return p, &buf
}

func TestShipIsReachableFromTownAndSailsBothWays(t *testing.T) {
	if exits[4]["east"] != zoneDocks || exits[zoneDocks]["west"] != 4 {
		t.Fatalf("the docks must hang off New Handington: %v %v", exits[4], exits[zoneDocks])
	}
	if exits[0]["up"] != 4 || exits[4]["down"] != 0 {
		t.Fatalf("town must be walkable from the Meadow")
	}
	p, buf := shipWorld(t, 50, 5000)
	cmdSail(p)
	p.w.Flush()
	if p.zoneID != zoneDeck || p.flow != 5000-voyageFare {
		t.Fatalf("boarding: zone %d flow %d\n%s", p.zoneID, p.flow, buf.String())
	}
	if !strings.Contains(buf.String(), "board the Open Balance") || !strings.Contains(buf.String(), "You smell:") {
		t.Errorf("deck should have text and a smell:\n%s", buf.String())
	}
	// not yet there
	voyageTick(time.Now().Add(voyageTime / 4))
	if p.zoneID != zoneDeck {
		t.Errorf("still at sea")
	}
	voyageTick(time.Now().Add(voyageTime/2 + time.Second))
	p.w.Flush()
	if !strings.Contains(buf.String(), "exactly halfway") {
		t.Errorf("expected the halfway message:\n%s", buf.String())
	}
	voyageTick(time.Now().Add(voyageTime + time.Second))
	p.w.Flush()
	if p.zoneID != zoneQuay || !strings.Contains(buf.String(), "Open Balance Quay") {
		t.Fatalf("should have landed at the Quay (zone %d)\n%s", p.zoneID, buf.String())
	}
	// and back
	cmdSail(p)
	voyageTick(time.Now().Add(voyageTime + time.Second))
	if p.zoneID != zoneDocks {
		t.Errorf("the return trip should land at the docks, zone %d", p.zoneID)
	}
}

func TestShipFareLevelWarningAndStrandProtection(t *testing.T) {
	poor, pbuf := shipWorld(t, 60, 100)
	cmdSail(poor)
	poor.w.Flush()
	if poor.zoneID != zoneDocks || !strings.Contains(pbuf.String(), "passage") {
		t.Errorf("too poor to sail:\n%s", pbuf.String())
	}
	low, lbuf := shipWorld(t, 10, 5000)
	cmdSail(low)
	low.w.Flush()
	if low.zoneID != zoneDocks || !strings.Contains(lbuf.String(), "level 35") || low.flow != 5000 {
		t.Errorf("a level 10 gets a warning first and is not charged:\n%s", lbuf.String())
	}
	cmdSail(low) // confirmed
	if low.zoneID != zoneDeck {
		t.Errorf("a second sail confirms the trip")
	}
	// logged out at sea (no voyage state): put back on the home docks
	stray, _ := shipWorld(t, 50, 0)
	stray.zoneID = zoneDeck
	stray.voyageTo = 0
	voyageTick(time.Now())
	if stray.zoneID != zoneDocks {
		t.Errorf("a stranded passenger should be returned to the docks, zone %d", stray.zoneID)
	}
	// no ship in the Meadow
	meadow, mbuf := shipWorld(t, 50, 5000)
	meadow.zoneID = 0
	cmdSail(meadow)
	meadow.w.Flush()
	if !strings.Contains(mbuf.String(), "no ship here") {
		t.Errorf("sail away from a port:\n%s", mbuf.String())
	}
}

// bfs distances from the Quay over the exit graph.
func quayDistances() map[int]int {
	dist := map[int]int{zoneQuay: 0}
	queue := []int{zoneQuay}
	for len(queue) > 0 {
		z := queue[0]
		queue = queue[1:]
		for dir, to := range exits[z] {
			if dir == "out" {
				continue
			}
			if _, seen := dist[to]; !seen {
				dist[to] = dist[z] + 1
				queue = append(queue, to)
			}
		}
	}
	return dist
}

var levelRe = regexp.MustCompile(`Recommended levels (\d+)-(\d+)`)

func zoneLevels(t *testing.T, zid int) (int, int) {
	m := levelRe.FindStringSubmatch(zoneDesc[zid])
	if m == nil {
		t.Fatalf("zone %d description has no level band", zid)
	}
	lo, _ := strconv.Atoi(m[1])
	hi, _ := strconv.Atoi(m[2])
	return lo, hi
}

func TestEveryReachZoneIsConnectedAndDifficultyClimbsWithDistance(t *testing.T) {
	dist := quayDistances()
	reach := []int{20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 401, 402, 403, 410, 411, 412, 421, 422, 423, 424, 425}
	for _, z := range reach {
		if _, ok := dist[z]; !ok {
			t.Errorf("zone %d (%s) cannot be reached from the Quay", z, owlZoneNames[z])
		}
	}
	// mean level of the overland zones by distance from the landing must not fall as you go out
	overland := []int{20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
	sums, counts := map[int]int{}, map[int]int{}
	for _, z := range overland {
		lo, _ := zoneLevels(t, z)
		sums[dist[z]] += lo
		counts[dist[z]]++
	}
	prev := 0
	for d := 0; d < 8; d++ {
		if counts[d] == 0 {
			continue
		}
		avg := sums[d] / counts[d]
		if avg < prev {
			t.Errorf("zones %d steps from the landing average Lv%d, easier than the ring before (Lv%d)", d, avg, prev)
		}
		prev = avg
	}
	// the landing is the gentlest, the Wellspring the hardest
	landLo, _ := zoneLevels(t, zoneQuay)
	topLo, _ := zoneLevels(t, 31)
	if topLo < landLo+40 {
		t.Errorf("capstone (Lv%d) should be far above the landing (Lv%d)", topLo, landLo)
	}
	// interior rooms climb toward their boss
	for _, chain := range [][]int{{401, 402, 403}, {410, 411, 412}, {421, 422, 423, 424, 425}} {
		last := 0
		for _, z := range chain {
			lo, _ := zoneLevels(t, z)
			if lo < last {
				t.Errorf("room %d (Lv%d) is easier than the room before it", z, lo)
			}
			last = lo
		}
		boss := mob.OwlZoneSpawns(chain[len(chain)-1])
		if len(boss) == 0 {
			t.Errorf("boss room %d spawns nothing", chain[len(chain)-1])
		}
	}
	// twelve named continent zones, three of them interior dungeons with several rooms
	named := map[int]bool{}
	for _, z := range reach {
		if z < 400 {
			named[z] = true
		}
	}
	if len(named) != 12 {
		t.Errorf("the Reach has %d named zones, want 12", len(named))
	}
}

func TestReachHasNPCsQuestsAndEveryZoneSmells(t *testing.T) {
	zonesWithQuests := map[int]bool{}
	for _, q := range questBank.All() {
		if strings.HasPrefix(q.ID, "reach-") {
			if n := npcByID(q.GiverNPCID); n != nil {
				zonesWithQuests[n.ZoneID] = true
			}
		}
	}
	for _, z := range []int{20, 21, 22, 23, 24, 25, 26, 28, 29, 30, 31} {
		if !zonesWithQuests[z] {
			t.Errorf("zone %d (%s) has no quests", z, owlZoneNames[z])
		}
		if !hasNPC(z) {
			t.Errorf("zone %d has no NPC", z)
		}
	}
	for _, z := range append([]int{19, 32}, mob.OwlZoneIDs()...) {
		if zoneSmell[z] == "" {
			t.Errorf("zone %d has no smell", z)
		}
	}
	// a nice, non-shop reward from the item database appears among the Reach quests
	nice := 0
	for _, q := range questBank.All() {
		switch q.RewardItem {
		case "fourth-remorse", "icewraith-claymore", "last-regret", "choker-of-dawn", "helm-of-the-forgotten-kingdom", "operator's-tunic", "bow-of-the-gorgon-outcast", "shepherd's-staff", "light's-end":
			nice++
		}
	}
	if nice < 6 {
		t.Errorf("expected several quests to reward end-game items from the database, found %d", nice)
	}
}
