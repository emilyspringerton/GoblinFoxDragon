package main

import "dragonsnshit/server/fame"

// NPCs for the starter zones and the town. Their quests live in server/quest/extra_quests.go.
func init() {
	npcs = append(npcs,
		npcDef{ID: "cave-cartographer-dunn", Name: "Cave Cartographer Dunn", ZoneID: 2, FameNation: fame.Neutral,
			Greeting: "Every map of these caves is wrong by a few feet, and the feet are always where you'd stand. Help me fix that and I'll make it worth your while."},
		npcDef{ID: "peat-cutter-odo", Name: "Peat Cutter Odo", ZoneID: 3, FameNation: fame.Neutral,
			Greeting: "Cut peat for forty winters. The bog keeps what it likes and gives back what it doesn't. Bring me peat and a bit of luck."},
		npcDef{ID: "mayor-pell", Name: "Mayor Pell", ZoneID: 4, FameNation: fame.Neutral,
			Greeting: "Welcome to New Handington. We have three rules: be polite, be brief, and don't document the pigeons. The rest is negotiable."},
	)
}
