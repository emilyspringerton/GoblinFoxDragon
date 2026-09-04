package mob

// ApplyVariant returns a real, standalone difficulty-tier copy of template (a mob returned by
// any existing *Spawns() constructor) -- same real position offset by a small, fixed amount so
// it renders as "right next to" the base mob it's templated from (GFD-MOBSPAWN-001 Phase 4,
// founder real-time: "harder mobs pretty close to lower level mobs with the same model and
// texture just a different name"), a new ID, DisplayName set to name, and HP/MaxHP/MeleeDamage
// scaled by mul -- the exact same real scaling shape dungeon.go's own DungeonEliteHPMul/
// DungeonBossHPMul already establishes for named dungeon bosses/elites, generalized here for any
// mob Kind. Kind itself is left untouched, so loot/quest/AI all still resolve to the real base
// mob this variant is templated from.
func ApplyVariant(template Mob, id, name string, mul float64) Mob {
	v := template
	v.ID = id
	v.DisplayName = name
	v.Pos.X += 3
	v.Pos.Z += 3
	v.HomePos = v.Pos
	v.HP = int(float64(template.HP) * mul)
	v.MaxHP = v.HP
	v.MeleeDamage = int(float64(template.MeleeDamage) * mul)
	return v
}
