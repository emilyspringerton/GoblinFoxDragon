package main

// hnm.go -- end-game "highly notorious" mobs: very strong named characters on ~24 hour respawns.
//
// Each HNM (hnm_gen.go, from scripts/gen_hnm.py) lives in one of the city-district zones, hits
// like a truck, resists crowd control, and drops the elite end-game gear from the item database.
// When one dies the whole server is told, and it returns 24 hours later plus up to two hours of
// jitter. The respawn time is saved to var/mud-hnm.json so a deploy or restart does not reset the
// wait (and does not let anyone farm one by bouncing the server).

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"dragonsnshit/server/mob"
)

type hnmDef struct {
	ID, Name string
	Zone     int
	Level    int
	HP       int
	Damage   int
	XP       int // fixed experience (a raw HP*10 award would be absurd)
	Blurb    string
}

var hnmByID = func() map[string]*hnmDef {
	m := map[string]*hnmDef{}
	for i := range hnmDefs {
		m[hnmDefs[i].ID] = &hnmDefs[i]
	}
	return m
}()

type hnmRecord struct {
	SpawnAt    int64  `json:"spawn_at"` // 0 = alive (or never killed)
	LastKill   int64  `json:"last_kill"`
	LastKiller string `json:"last_killer"`
}

type hnmStore struct {
	path string
	mu   sync.Mutex
	data map[string]*hnmRecord
}

var mudHNM = newHNMStore("var/mud-hnm.json")

func newHNMStore(path string) *hnmStore {
	s := &hnmStore{path: path, data: map[string]*hnmRecord{}}
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &s.data)
	}
	return s
}

func (s *hnmStore) get(id string) hnmRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.data[id]; ok {
		return *r
	}
	return hnmRecord{}
}

func (s *hnmStore) set(id string, r hnmRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[id] = &r
	raw, err := json.Marshal(s.data)
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0755)
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, raw, 0644) == nil {
		_ = os.Rename(tmp, s.path)
	}
}

func hnmMob(d *hnmDef) mob.Mob {
	pos := mob.Pos{X: 0, Y: 1, Z: -8}
	return mob.Mob{
		ID: d.ID, Kind: d.Name, SceneID: d.Zone, Pos: pos, HomePos: pos,
		HP: d.HP, MaxHP: d.HP, State: mob.StateIdle,
		AggroRange: 8, LeashRange: 45, MeleeRange: 3.5, MoveSpeed: 4.2,
		SwingDelay: 2400 * time.Millisecond, MeleeDamage: d.Damage, CCResist: 0.85,
	}
}

func hnmAlive(d *hnmDef) bool {
	reg := gw.mobRegs[d.Zone]
	if reg == nil {
		return false
	}
	m, ok := reg.Get(d.ID)
	return ok && m.HP > 0
}

// hnmSpawn puts the HNM into its lair (reviving it if its corpse is still registered).
func hnmSpawn(d *hnmDef) {
	reg := gw.mobRegs[d.Zone]
	if reg == nil {
		reg = mob.New()
		gw.mobRegs[d.Zone] = reg
	}
	if _, ok := reg.Get(d.ID); ok {
		_ = reg.Revive(d.ID, hnmMob(d).Pos)
		return
	}
	_ = reg.Spawn(hnmMob(d))
}

// hnmInit is called at world start: anything whose timer has run out (or that was never killed) is
// spawned; anything still on cooldown stays away until its saved time.
func hnmInit(now time.Time) {
	for i := range hnmDefs {
		d := &hnmDefs[i]
		if gw.mobRegs[d.Zone] == nil {
			gw.mobRegs[d.Zone] = mob.New()
		}
		rec := mudHNM.get(d.ID)
		if rec.SpawnAt == 0 || now.Unix() >= rec.SpawnAt {
			hnmSpawn(d)
			rec.SpawnAt = 0
			mudHNM.set(d.ID, rec)
		}
	}
}

// hnmTick spawns any HNM whose timer has expired.
func hnmTick(now time.Time) {
	for i := range hnmDefs {
		d := &hnmDefs[i]
		rec := mudHNM.get(d.ID)
		if rec.SpawnAt == 0 || now.Unix() < rec.SpawnAt || hnmAlive(d) {
			continue
		}
		hnmSpawn(d)
		rec.SpawnAt = 0
		mudHNM.set(d.ID, rec)
		broadcastAll(fmt.Sprintf("[!!!] %s has awoken in %s! (Lv %d)", d.Name, zoneName(d.Zone), d.Level))
	}
}

// hnmKilled records the kill, schedules the next spawn and tells everyone.
func hnmKilled(d *hnmDef, killer string, now time.Time) {
	jitter := time.Duration(rand.Int63n(int64(hnmJitterHours) * int64(time.Hour)))
	next := now.Add(hnmRespawnHours*time.Hour + jitter)
	mudHNM.set(d.ID, hnmRecord{SpawnAt: next.Unix(), LastKill: now.Unix(), LastKiller: killer})
	broadcastAll(fmt.Sprintf("[!!!] %s has been slain in %s by %s! It will not return for a day.", d.Name, zoneName(d.Zone), killer))
}

func cmdHNM(p *player) {
	now := time.Now()
	p.send("\r\n=== Notorious monsters (24 hour respawn) ===")
	defs := append([]hnmDef(nil), hnmDefs...)
	sort.Slice(defs, func(i, j int) bool { return defs[i].Level < defs[j].Level })
	for i := range defs {
		d := &defs[i]
		status := "ALIVE"
		if !hnmAlive(d) {
			rec := mudHNM.get(d.ID)
			if rec.SpawnAt > now.Unix() {
				left := time.Until(time.Unix(rec.SpawnAt, 0)).Round(time.Minute)
				status = fmt.Sprintf("respawns in about %s", left)
			} else {
				status = "stirring..."
			}
		}
		p.sendf("  Lv%-3d %-42s %-26s %s", d.Level, d.Name, zoneName(d.Zone), status)
		if hnmAlive(d) {
			p.sendf("        %s", d.Blurb)
		}
	}
	p.prompt()
}

func isHNM(id string) bool { _, ok := hnmByID[id]; return ok }

func hnmListLine(m *mob.Mob) string {
	if d, ok := hnmByID[m.ID]; ok {
		return fmt.Sprintf(" [NOTORIOUS Lv%d]", d.Level)
	}
	return ""
}

var _ = strings.TrimSpace
