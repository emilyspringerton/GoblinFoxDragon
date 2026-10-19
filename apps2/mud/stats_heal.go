package main

// healPower scales a heal's base amount by the caster's MND (gear included): every point of MND
// above 10 adds 2 HP, so a healer with gear really does heal more.
func healPower(p *player, base int) int {
	if mnd := playerCombatStats(p).MND; mnd > 10 {
		return base + (mnd-10)*2
	}
	return base
}
