package fame

// trapx_factions.go: TRAPX faction name overlay (S122-05).
//
// TRAPX uses the same Nation constants as DragonsNShit but maps them to
// real city factions instead of fantasy city-states:
//
//   Sandoria  → The Frequency (signal, code, CAST terminals, information flow)
//   Bastok    → The Bloc (social cohesion, community mutual aid, FO defense)
//   Windurst  → Procurement Houses (economic leverage, Coverage contracts, media deals)
//
// TRAPXFactionName is used by the MUD `fame` command when the active world
// context is TRAPX (player is in district scenes 200–299).

var trapxFactionNames = map[Nation]string{
	Neutral:  "Unaligned",
	Sandoria: "The Frequency",
	Bastok:   "The Bloc",
	Windurst: "Procurement Houses",
}

// TRAPXFactionName returns the TRAPX faction name for a Nation constant.
func TRAPXFactionName(n Nation) string {
	if s, ok := trapxFactionNames[n]; ok {
		return s
	}
	return "Unknown"
}

// TRAPXFactionDesc returns a one-line description of the faction.
func TRAPXFactionDesc(n Nation) string {
	switch n {
	case Sandoria:
		return "The Frequency: signal network, CAST terminal access, code distribution. Information is power."
	case Bastok:
		return "The Bloc: community solidarity, mutual aid, FO defense networks. The block protects its own."
	case Windurst:
		return "Procurement Houses: Coverage contracts, media deals, economic leverage. Everything has a price."
	default:
		return "No faction alignment."
	}
}

// TRAPXFactionBenefit describes what high rank grants per faction.
func TRAPXFactionBenefit(n Nation, rank int) string {
	switch n {
	case Sandoria: // The Frequency
		switch {
		case rank >= 5:
			return "CAST terminal city-wide access; Frequency Ghost sigil available"
		case rank >= 3:
			return "Coverage contract discount 25%; CAST fragment delivery"
		default:
			return "CAST terminal local access"
		}
	case Bastok: // The Bloc
		switch {
		case rank >= 5:
			return "Block Patrol: nearby NPCs defend FOs you hold; Bloc NPCs patrol your district"
		case rank >= 3:
			return "FO flip window extended +1 flip/day; community tip-offs (NPC alerts you to contester)"
		default:
			return "Trust bonus: +5 Watcher Trust on entering a Bloc-friendly district"
		}
	case Windurst: // Procurement Houses
		switch {
		case rank >= 5:
			return "Coverage deal tier 3: media spike costs reduced 50%; Vendor Overseer ally in T3 events"
		case rank >= 3:
			return "Procurement discount: K9 battery + audit-receipt items cost 25% less at vendors"
		default:
			return "Receipt markup: sell receipts to Procurement Houses at +10% value"
		}
	default:
		return "No benefit."
	}
}

// TRAPXFactions returns the 3 TRAPX playable factions in canonical order.
func TRAPXFactions() []Nation {
	return []Nation{Sandoria, Bastok, Windurst}
}
