package mob

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// Eastwind Owl zone cluster (zones 10-18). The tables live in owl_gen.go, generated from
// scripts/gen_owl_zones.py; this file is the hand-written consumer. Kinds are plain base mobs
// (HP/damage rising zone over zone); the "harder neighbours" come from data/mob_variants.json and
// data/mob_spawns.json, the same files the IDUNA mob-spawn admin tools edit.

// OwlMobDef is one base mob kind in an Owl zone's spawn ring layout.
type OwlMobDef struct {
	Kind                               string
	HP, Damage                         int
	MoveSpeed                          float64
	AggroRange, LeashRange, MeleeRange float64
	SwingMS                            int
	Count                              int     // how many are placed
	Ring                               float64 // radius of the ring they are spread on
}

// OwlNMDef is a notorious monster template for an Owl zone. Kind doubles as the drop-table key
// (lower-cased by mobdrop), so it is the player-facing name.
type OwlNMDef struct {
	ID, Kind                           string
	ZoneID                             int
	HP, Damage                         int
	AggroRange, LeashRange, MeleeRange float64
	MoveSpeed                          float64
	SwingMS                            int
	X, Z                               float64
}

// OwlZoneIDs returns the Owl cluster's zone IDs in ascending order.
func OwlZoneIDs() []int {
	ids := make([]int, 0, len(OwlZoneMobs))
	for id := range OwlZoneMobs {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// OwlZoneSpawns lays each kind of the zone out on its ring. Mob IDs are "<kind>-z<zone>-<n>", so
// "<kind>-z<zone>-0" is the stable placeholder ID for that kind.
func OwlZoneSpawns(zoneID int) []Mob {
	var out []Mob
	for ki, d := range OwlZoneMobs[zoneID] {
		for i := 0; i < d.Count; i++ {
			ang := 2*math.Pi*float64(i)/float64(d.Count) + float64(ki)*0.7
			pos := Pos{X: d.Ring * math.Cos(ang), Y: 1, Z: d.Ring * math.Sin(ang)}
			out = append(out, Mob{
				ID:          fmt.Sprintf("%s-z%d-%d", d.Kind, zoneID, i),
				Kind:        d.Kind,
				SceneID:     zoneID,
				Pos:         pos,
				HomePos:     pos,
				HP:          d.HP,
				MaxHP:       d.HP,
				State:       StateIdle,
				AggroRange:  d.AggroRange,
				LeashRange:  d.LeashRange,
				MeleeRange:  d.MeleeRange,
				MoveSpeed:   d.MoveSpeed,
				SwingDelay:  time.Duration(d.SwingMS) * time.Millisecond,
				MeleeDamage: d.Damage,
			})
		}
	}
	return out
}

// OwlNMTemplate returns the spawnable Mob for an Owl-cluster NM ID.
func OwlNMTemplate(nmID string) (Mob, bool) {
	for _, d := range OwlNMDefs {
		if d.ID != nmID {
			continue
		}
		pos := Pos{X: d.X, Y: 1, Z: d.Z}
		return Mob{
			ID: d.ID, Kind: d.Kind, SceneID: d.ZoneID, Pos: pos, HomePos: pos,
			HP: d.HP, MaxHP: d.HP, State: StateIdle,
			AggroRange: d.AggroRange, LeashRange: d.LeashRange, MeleeRange: d.MeleeRange,
			MoveSpeed: d.MoveSpeed, SwingDelay: time.Duration(d.SwingMS) * time.Millisecond,
			MeleeDamage: d.Damage,
		}, true
	}
	return Mob{}, false
}
