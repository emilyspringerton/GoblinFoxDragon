package mob

import "testing"

func TestOwlZoneDifficultyRisesAlongBranches(t *testing.T) {
	maxHP := func(zid int) int {
		m := 0
		for _, d := range OwlZoneMobs[zid] {
			if d.HP > m {
				m = d.HP
			}
		}
		return m
	}
	// Each zone is deeper than the one it branches from.
	for _, pair := range [][2]int{{10, 14}, {11, 15}, {12, 16}, {13, 17}, {14, 18}} {
		if maxHP(pair[0]) >= maxHP(pair[1]) {
			t.Errorf("zone %d should be easier than %d", pair[0], pair[1])
		}
	}
}

func TestOwlZoneSpawnsMixTiersWithinZone(t *testing.T) {
	for _, zid := range OwlZoneIDs() {
		defs := OwlZoneMobs[zid]
		if len(defs) < 3 || defs[0].HP >= defs[len(defs)-1].HP {
			t.Errorf("zone %d should have an easy->hard mob spread", zid)
		}
		if len(OwlZoneSpawns(zid)) == 0 {
			t.Errorf("zone %d spawns nothing", zid)
		}
	}
}

func TestOwlNMsOutclassTheirZone(t *testing.T) {
	for _, nm := range OwlNMDefs {
		for _, d := range OwlZoneMobs[nm.ZoneID] {
			if nm.HP <= d.HP*2 {
				t.Errorf("%s (%d HP) should dwarf %s (%d HP)", nm.ID, nm.HP, d.Kind, d.HP)
			}
		}
	}
}
