package quest

// trapx_chains.go: 8 TRAPX RPG class unlock quest chains (S122-04).
//
// Each chain unlocks a quest-gated TRAPX job class. Completing the final quest
// in a chain rewards a "job-stone-<JOB>" item, which the MUD setjob command
// consumes to unlock the class permanently on the character.
//
// Chain overview:
//   DRK — 3 moral-ambiguity quests (field extraction in contested zones)
//   BST — 3 K9 Doctrine quests (deploy + manage swarms, pass Tier 3 tech check)
//   BRD — 3 Media faction access quests (Channel 11 broadcast + coverage deals)
//   SAM — 3 precision combat trial quests (flip under pressure, clean reads)
//   SMN — 7 city-entity encounter quests (meet all 7 SMN entities, Baphomet first)
//   BLU — 3 mob encounter absorption quests (15 unique mob kinds observed)
//   GEO — 2 district mapping quests (map all 5 district types)
//   RUN — 3 Oversight Sect boss quests (face the institutional threat arc)

// TRAPXJobChains contains all 8 job unlock quest chains.
// Each slice is ordered (chain step 1 → final unlock).
var TRAPXJobChains = map[string][]*Quest{
	"DRK": darkKnightChain(),
	"BST": beastmasterChain(),
	"BRD": bardChain(),
	"SAM": samuraiChain(),
	"SMN": summonerChain(),
	"BLU": blueChain(),
	"GEO": geomancerChain(),
	"RUN": wardRunnerChain(),
}

// TRAPXChainQuests returns all TRAPX chain quests as a flat slice for loading into a Bank.
func TRAPXChainQuests() []*Quest {
	var out []*Quest
	for _, chain := range TRAPXJobChains {
		out = append(out, chain...)
	}
	return out
}

// ── DRK: Dark Knight — moral-ambiguity arc ────────────────────────────────────

func darkKnightChain() []*Quest {
	return []*Quest{
		{
			ID:         "drk-01-extraction",
			Title:      "The Wrong Side of the Receipt",
			Desc:       "Extract Flow from an FO currently held by a rival crew. You will be seen. The ledger will remember.",
			GiverNPCID: "scar-keeper",
			RequireKills: map[string]int{"guard": 1},
			RewardGil:  0,
			RewardItem: "",
			RewardFame: 0,
		},
		{
			ID:         "drk-02-contested",
			Title:      "Contest in the Rain",
			Desc:       "Initiate a contest on a held FO when Enforcement is at Level 2 or higher. Finish the window.",
			GiverNPCID: "scar-keeper",
			RequireKills: map[string]int{"enforcer": 2},
			RewardGil:  200,
			RewardItem: "",
			RewardFame: 0,
		},
		{
			ID:         "drk-03-judgement",
			Title:      "What the Ledger Knows",
			Desc:       "Flip an FO from Contested to Held. A Rogue Swarm begins. Survive containment without Bird Correction.",
			GiverNPCID: "scar-keeper",
			RequireKills: map[string]int{"rogue-dog": 5},
			RewardGil:  500,
			RewardItem: "job-stone-drk",
			RewardFame: 10,
		},
	}
}

// ── BST: Beastmaster / K9 Handler — Tier 3 tech doctrine ────────────────────

func beastmasterChain() []*Quest {
	return []*Quest{
		{
			ID:         "bst-01-doctrine",
			Title:      "K9 Doctrine: First Deployment",
			Desc:       "Deploy a 2-dog Sentry swarm on a held FO and sustain it for one full tick.",
			GiverNPCID: "warehouse-contact",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"k9-battery": 2},
			RewardGil:  100,
			RewardItem: "",
			RewardFame: 0,
		},
		{
			ID:         "bst-02-audit-mode",
			Title:      "K9 Doctrine: Audit Protocol",
			Desc:       "Switch a deployed dog to Audit mode. Generate 3 receipts from the swarm in one session.",
			GiverNPCID: "warehouse-contact",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"audit-receipt": 3},
			RewardGil:  250,
			RewardItem: "",
			RewardFame: 0,
		},
		{
			ID:         "bst-03-tier3",
			Title:      "K9 Doctrine: Tier 3 Access",
			Desc:       "Reach Tech Pressure Tier 3 (QuietAudit: 650). Manage your swarm through the audit window without losing dogs to rogue death.",
			GiverNPCID: "warehouse-contact",
			RequireKills: map[string]int{"audit-drone": 3},
			RewardGil:  600,
			RewardItem: "job-stone-bst",
			RewardFame: 15,
		},
	}
}

// ── BRD: Bard / Broadcaster — Media faction access ────────────────────────────

func bardChain() []*Quest {
	return []*Quest{
		{
			ID:         "brd-01-channel11",
			Title:      "Channel 11 Access",
			Desc:       "Talk to the Broadcast Operator. Hold an FO when Attention exceeds 500 — Channel 11 cameras are rolling.",
			GiverNPCID: "broadcast-operator",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"broadcast-pass": 1},
			RewardGil:  0,
			RewardItem: "broadcast-pass",
			RewardFame: 5,
		},
		{
			ID:         "brd-02-coverage",
			Title:      "Coverage Deal",
			Desc:       "Reach a Coverage deal with The Frequency. Deliver 5 city receipts to the broadcast-operator NPC within 10 minutes.",
			GiverNPCID: "broadcast-operator",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"city-receipt": 5},
			RewardGil:  300,
			RewardItem: "",
			RewardFame: 10,
		},
		{
			ID:         "brd-03-narrative",
			Title:      "The Story We Tell",
			Desc:       "Trigger a MYTH_SEEDED event in any district by raising Fear above 90. The Broadcast Operator will interview you.",
			GiverNPCID: "broadcast-operator",
			RequireKills: map[string]int{"watcher-camera": 1},
			RewardGil:  400,
			RewardItem: "job-stone-brd",
			RewardFame: 20,
		},
	}
}

// ── SAM: Samurai / Blade Runner — precision combat trial ─────────────────────

func samuraiChain() []*Quest {
	return []*Quest{
		{
			ID:         "sam-01-clean-read",
			Title:      "The Clean Read",
			Desc:       "Flip an FO without triggering an ALERT_SPIKE event. Read the district. Wait for the window.",
			GiverNPCID: "corner-kid",
			RequireKills: map[string]int{},
			RewardGil:  0,
			RewardItem: "",
			RewardFame: 0,
		},
		{
			ID:         "sam-02-pressure",
			Title:      "Under Pressure",
			Desc:       "Flip an FO while Enforcement is at Level 3 (Heavy). You have 90 seconds from Contested to Flipped.",
			GiverNPCID: "corner-kid",
			RequireKills: map[string]int{"enforcer": 4},
			RewardGil:  350,
			RewardItem: "",
			RewardFame: 0,
		},
		{
			ID:         "sam-03-trial",
			Title:      "The Blade Runner Trial",
			Desc:       "Complete 3 FO flips in a single session across different districts. No contester defeats.",
			GiverNPCID: "corner-kid",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"flip-receipt": 3},
			RewardGil:  700,
			RewardItem: "job-stone-sam",
			RewardFame: 15,
		},
	}
}

// ── SMN: Summoner / Avatar Caller — 7 city-entity encounter quests ────────────

func summonerChain() []*Quest {
	// 7 encounters, one per SMN entity avatar.
	return []*Quest{
		{
			ID:         "smn-01-baphomet",
			Title:      "The First Sigil: Baphomet",
			Desc:       "Enter the Abandoned district (scene 204) and find Baphomet's sigil. Survive 10 minutes in the district.",
			GiverNPCID: "scar-keeper",
			RequireKills: map[string]int{"abandoned-ghost": 2},
			RewardGil:  0,
			RewardItem: "sigil-baphomet",
			RewardFame: 5,
		},
		{
			ID:         "smn-02-anchor",
			Title:      "The Second Sigil: The Anchor",
			Desc:       "Hold an Industrial district FO for 5 consecutive minutes. The Anchor watches through the crane cameras.",
			GiverNPCID: "warehouse-contact",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"sigil-baphomet": 1},
			RewardGil:  100,
			RewardItem: "sigil-anchor",
			RewardFame: 5,
		},
		{
			ID:         "smn-03-frequency-ghost",
			Title:      "The Third Sigil: Frequency Ghost",
			Desc:       "Access a CAST terminal in the Underground and read 3 lore fragments. The Frequency Ghost is in the signal.",
			GiverNPCID: "frequency-runner",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"sigil-anchor": 1, "cast-fragment": 3},
			RewardGil:  150,
			RewardItem: "sigil-frequency-ghost",
			RewardFame: 5,
		},
		{
			ID:         "smn-04-warden",
			Title:      "The Fourth Sigil: The Warden",
			Desc:       "Survive Enforcement Level 4 (Lockdown) in the Commercial district for 5 minutes. The Warden sees only containment.",
			GiverNPCID: "broadcast-operator",
			RequireKills: map[string]int{"lockdown-cop": 8},
			RequireItems: map[string]int{"sigil-frequency-ghost": 1},
			RewardGil:  200,
			RewardItem: "sigil-warden",
			RewardFame: 5,
		},
		{
			ID:         "smn-05-smoke-tongue",
			Title:      "The Fifth Sigil: Smoke Tongue",
			Desc:       "Navigate the Underground while invisible (SNK + INV active). Do not trigger Watcher alertness. Reach the exit.",
			GiverNPCID: "frequency-runner",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"sigil-warden": 1},
			RewardGil:  250,
			RewardItem: "sigil-smoke-tongue",
			RewardFame: 5,
		},
		{
			ID:         "smn-06-static-king",
			Title:      "The Sixth Sigil: Static King",
			Desc:       "Hold an FO across all 5 districts simultaneously. City-wide. The Static King only recognises total presence.",
			GiverNPCID: "pawn-shop-runner",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"sigil-smoke-tongue": 1},
			RewardGil:  500,
			RewardItem: "sigil-static-king",
			RewardFame: 10,
		},
		{
			ID:         "smn-07-dragon",
			Title:      "The Seventh Sigil: The Dragon",
			Desc:       "Witness a Dragon-triggered city event (Emily Prime fires a Rogue Swarm or media spike). Be in the affected district when it fires. The Dragon acknowledges those who watch.",
			GiverNPCID: "scar-keeper",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"sigil-static-king": 1, "dragon-witness-receipt": 1},
			RewardGil:  1000,
			RewardItem: "job-stone-smn",
			RewardFame: 30,
		},
	}
}

// ── BLU: Blue Mage / Absorber — 15 unique mob encounter quests ────────────────

func blueChain() []*Quest {
	return []*Quest{
		{
			ID:         "blu-01-first-five",
			Title:      "Absorption Log: 1-5",
			Desc:       "Observe and engage 5 unique mob kinds across the city districts. Record their patterns.",
			GiverNPCID: "minibike-rider",
			RequireKills: map[string]int{"guard": 1, "enforcer": 1, "rogue-dog": 1, "watcher-camera": 1, "abandoned-ghost": 1},
			RewardGil:  100,
			RewardItem: "absorption-log-1",
			RewardFame: 0,
		},
		{
			ID:         "blu-02-second-five",
			Title:      "Absorption Log: 6-10",
			Desc:       "Engage 5 more unique mob kinds. Include at least one from each of the 5 city districts.",
			GiverNPCID: "minibike-rider",
			RequireItems: map[string]int{"absorption-log-1": 1},
			RequireKills: map[string]int{"audit-drone": 1, "lockdown-cop": 1, "coverage-drone": 1, "rogue-swarm-alpha": 1, "tech-hound": 1},
			RewardGil:  200,
			RewardItem: "absorption-log-2",
			RewardFame: 0,
		},
		{
			ID:         "blu-03-final-five",
			Title:      "Absorption Complete: 15 Unique",
			Desc:       "Absorb patterns from 5 rare mob kinds — including the Rogue Swarm Alpha and the Tech Hound during a Packmind event. The Absorber learns from the city's worst moments.",
			GiverNPCID: "minibike-rider",
			RequireItems: map[string]int{"absorption-log-2": 1},
			RequireKills: map[string]int{"packmind-alpha": 1, "crown-sentinel": 1, "signal-ghost": 1, "jiangshi-observer": 1, "dragon-echo": 1},
			RewardGil:  800,
			RewardItem: "job-stone-blu",
			RewardFame: 20,
		},
	}
}

// ── GEO: Geomancer / City Reader — district mapping ───────────────────────────

func geomancerChain() []*Quest {
	return []*Quest{
		{
			ID:         "geo-01-survey",
			Title:      "District Survey: The 5 Faces",
			Desc:       "Visit all 5 city districts (200–204). Talk to one NPC in each. The city reveals itself to those who walk it.",
			GiverNPCID: "minibike-rider",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"district-stamp-200": 1, "district-stamp-201": 1, "district-stamp-202": 1, "district-stamp-203": 1, "district-stamp-204": 1},
			RewardGil:  300,
			RewardItem: "",
			RewardFame: 5,
		},
		{
			ID:         "geo-02-reader",
			Title:      "City Reader Certification",
			Desc:       "Read the Watcher alertness in all 5 districts in a single session. Identify which district is most distressed. Report to the Scar Keeper.",
			GiverNPCID: "scar-keeper",
			RequireKills: map[string]int{},
			RequireItems: map[string]int{"district-report": 1},
			RewardGil:  500,
			RewardItem: "job-stone-geo",
			RewardFame: 10,
		},
	}
}

// ── RUN: Rune Fencer / Ward Runner — Oversight Sect boss arc ─────────────────

func wardRunnerChain() []*Quest {
	return []*Quest{
		{
			ID:         "run-01-sector-boss",
			Title:      "Ward Runner: Sector Boss",
			Desc:       "Defeat the Enforcement Sector Boss — a Level 3 Enforcer Commander who spawns at Lockdown. Available in any district at Level 4.",
			GiverNPCID: "corner-kid",
			RequireKills: map[string]int{"enforcer-commander": 1},
			RewardGil:  400,
			RewardItem: "oversight-badge",
			RewardFame: 10,
		},
		{
			ID:         "run-02-audit-boss",
			Title:      "Ward Runner: The Quiet Audit",
			Desc:       "Face the Quiet Audit Boss — a Vendor Overseer who spawns when Tech Pressure reaches T3. Defeat it before it generates a Crown Protocol event.",
			GiverNPCID: "warehouse-contact",
			RequireKills: map[string]int{"vendor-overseer": 1},
			RequireItems: map[string]int{"oversight-badge": 1},
			RewardGil:  600,
			RewardItem: "oversight-badge-2",
			RewardFame: 15,
		},
		{
			ID:         "run-03-crown",
			Title:      "Ward Runner: Crown Protocol",
			Desc:       "Survive the Crown Protocol event (Tech Pressure at T5: 1000). The city is at its worst. The Ward Runner is its last line. Reach the FO and defend it through the 60-second Crown window.",
			GiverNPCID: "scar-keeper",
			RequireKills: map[string]int{"crown-sentinel": 3},
			RequireItems: map[string]int{"oversight-badge-2": 1},
			RewardGil:  1000,
			RewardItem: "job-stone-run",
			RewardFame: 25,
		},
	}
}
