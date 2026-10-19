package main

import (
	"bufio"
	"bytes"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dragonsnshit/server/enmity"
	"dragonsnshit/server/homepoint"
	"dragonsnshit/server/mob"
	"dragonsnshit/server/mobdrop"
)

func hnmWorld(t *testing.T) (*player, *bytes.Buffer) {
	t.Helper()
	oldStore, oldGW := mudHNM, gw
	mudHNM = newHNMStore(filepath.Join(t.TempDir(), "hnm.json"))
	gw = &world{mobRegs: map[int]*mob.Registry{}, mobEnmity: map[string]*enmity.Table{}, players: map[string]*player{},
		charIDBySlot: map[string]string{}, rng: rand.New(rand.NewSource(1))}
	t.Cleanup(func() { mudHNM, gw = oldStore, oldGW })
	p := newTestPlayer()
	p.slot, p.name, p.zoneID = "hero", "Hero", 207
	p.homePoint = homepoint.NewState(207)
	p.combat = &mob.PlayerCombat{}
	var buf bytes.Buffer
	p.w = bufio.NewWriter(&buf)
	gw.players["hero"] = p
	return p, &buf
}

func TestHNMRosterIsEndgameAndDropsTheEliteItems(t *testing.T) {
	if len(hnmDefs) < 8 {
		t.Fatalf("expected a roster of end-game mobs, got %d", len(hnmDefs))
	}
	reg := mobdrop.NewRegistry()
	if err := reg.LoadFile("../../data/mob_drops.json"); err != nil {
		t.Fatal(err)
	}
	items := map[string]bool{}
	for _, it := range loadRawItems(t) {
		items[slugOf(it.Name)] = true
	}
	seenSoul, seenKatana := false, false
	for _, d := range hnmDefs {
		if d.Level < 80 || d.HP < 50000 || d.Damage < 200 {
			t.Errorf("%s is not end-game strong enough (Lv%d HP%d dmg%d)", d.Name, d.Level, d.HP, d.Damage)
		}
		if strings.HasPrefix(d.Name, "nm-") || strings.Contains(d.Name, "ARENA_") {
			t.Errorf("HNM %q should have a character name", d.Name)
		}
		if !strings.HasPrefix(d.ID, "nm-hnm-") {
			t.Errorf("%s: id should keep the nm- prefix for NM behaviour (CC resist, crisis)", d.ID)
		}
		drops := reg.DropsFor(d.Name)
		if len(drops) < 5 {
			t.Errorf("%s has only %d drops", d.Name, len(drops))
		}
		for _, dr := range drops {
			if !items[dr.ID] {
				t.Errorf("%s drops %q which is not an item", d.Name, dr.ID)
			}
			if dr.ID == "soul-of-jordan" {
				seenSoul = true
			}
			if dr.ID == "onimaru" || dr.ID == "mikazuki" {
				seenKatana = true
			}
		}
		if !gearZoneOK(d.Zone) {
			t.Errorf("%s lair zone %d is not a known zone", d.Name, d.Zone)
		}
	}
	if !seenSoul || !seenKatana {
		t.Errorf("expected Soul of Jordan (%v) and the end-game katanas (%v) among HNM drops", seenSoul, seenKatana)
	}
}

func gearZoneOK(z int) bool { _, ok := zoneDesc[z]; return ok }

func TestHNMSpawnsOnFirstBootAndNeverOnTheShortRespawnTimer(t *testing.T) {
	p, _ := hnmWorld(t)
	now := time.Now()
	hnmInit(now)
	for i := range hnmDefs {
		if !hnmAlive(&hnmDefs[i]) {
			t.Fatalf("%s should be alive on first boot", hnmDefs[i].Name)
		}
	}
	d := &hnmDefs[len(hnmDefs)-1] // No-Name, zone 207
	reg := gw.mobRegs[d.Zone]
	m, _ := reg.Get(d.ID)
	if m.CCResist < 0.8 {
		t.Errorf("an HNM resists crowd control (%v)", m.CCResist)
	}
	// kill it through the real kill path
	p.zoneID = d.Zone
	p.questJournal = nil
	m.HP = 0
	m.State = mob.StateDead
	before := len(gw.deadQueue)
	hnmKilled(d, p.name, now)
	if len(gw.deadQueue) != before {
		t.Errorf("HNMs must not go on the normal short respawn queue")
	}
	rec := mudHNM.get(d.ID)
	wait := time.Unix(rec.SpawnAt, 0).Sub(now)
	if wait < 24*time.Hour || wait > 26*time.Hour+time.Minute {
		t.Errorf("respawn in %v, want 24h plus up to 2h of jitter", wait)
	}
	// an hour later: still gone; a day and two hours later: back
	hnmTick(now.Add(time.Hour))
	if hnmAlive(d) {
		t.Errorf("must not return after an hour")
	}
	hnmTick(now.Add(26*time.Hour + 2*time.Minute))
	if !hnmAlive(d) {
		t.Errorf("must be back after the window")
	}
	if got, _ := reg.Get(d.ID); got.HP != got.MaxHP {
		t.Errorf("respawn must be at full health")
	}
}

func TestHNMTimerSurvivesARestart(t *testing.T) {
	_, _ = hnmWorld(t)
	d := &hnmDefs[0]
	now := time.Now()
	hnmInit(now)
	hnmKilled(d, "Someone", now)
	path := mudHNM.path
	// "restart": new store from the same file, fresh world
	mudHNM = newHNMStore(path)
	gw.mobRegs = map[int]*mob.Registry{}
	hnmInit(now.Add(time.Hour))
	if hnmAlive(d) {
		t.Fatalf("a restart one hour after the kill must not respawn it (deploys would otherwise reset 24h timers)")
	}
	for i := 1; i < len(hnmDefs); i++ {
		if !hnmAlive(&hnmDefs[i]) {
			t.Errorf("%s was never killed and should be up after restart", hnmDefs[i].Name)
		}
	}
	hnmInit(now.Add(27 * time.Hour))
	if !hnmAlive(d) {
		t.Errorf("after the window a restart spawns it")
	}
}

func TestHNMKillAnnouncesAndAwardsFixedXP(t *testing.T) {
	p, buf := hnmWorld(t)
	hnmInit(time.Now())
	hnmKilled(&hnmDefs[0], p.name, time.Now())
	p.w.Flush()
	if !strings.Contains(buf.String(), "has been slain") || !strings.Contains(buf.String(), hnmDefs[0].Name) {
		t.Errorf("the whole server should be told:\n%s", buf.String())
	}
	if hnmDefs[0].XP <= 0 || hnmDefs[0].XP >= hnmDefs[0].HP*10 {
		t.Errorf("XP should be a fixed sane award, got %d (HP*10 would be %d)", hnmDefs[0].XP, hnmDefs[0].HP*10)
	}
}

func TestHNMCommandListsStatus(t *testing.T) {
	p, buf := hnmWorld(t)
	hnmInit(time.Now())
	if m, ok := gw.mobRegs[hnmDefs[2].Zone].Get(hnmDefs[2].ID); ok {
		m.HP, m.State = 0, mob.StateDead
	}
	hnmKilled(&hnmDefs[2], "Hero", time.Now())
	cmdHNM(p)
	p.w.Flush()
	out := buf.String()
	if !strings.Contains(out, "ALIVE") || !strings.Contains(out, "respawns in about") {
		t.Errorf("hnm list should show both states:\n%s", out)
	}
	if displayMobID("nm-hnm-no-name") == "nm-hnm-no-name" || strings.HasPrefix(displayMobID("nm-hnm-x"), "nm-") {
		t.Errorf("displayMobID leaks the prefix: %q", displayMobID("nm-hnm-x"))
	}
}
