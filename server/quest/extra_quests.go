package quest

// Quests for the starter zones, the town and the city districts. Rewards are real rows of
// data/items.json (apps2/mud/quest_content_test.go checks that), and several chains ask for crafted
// goods so the new crafting system has a reason to be used from the very first zone.
var ExtraQuests = []*Quest{
	// ---- Meadow: the Village Elder
	{ID: "start-elder-worms", Title: "Worm Hunt", Desc: "The Village Elder wants 8 worms cleared from the meadow so the children can play.",
		GiverNPCID: "elder", RequireKills: map[string]int{"worm": 8}, RewardFlow: 150, RewardItem: "iron-shortsword"},
	{ID: "start-elder-ingots", Title: "A Smith's Errand", Desc: "Bring the Elder 3 Copper Ingots -- smelt them at the forge with 'craft copper-ingot'.",
		GiverNPCID: "elder", RequireItems: map[string]int{"copper-ingot": 3}, RewardFlow: 250, RewardItem: "copper-knife"},
	{ID: "start-elder-fish", Title: "Fish Supper", Desc: "The Elder is hungry for fish. Bring 2 Carp from the meadow stream.",
		GiverNPCID: "elder", RequireItems: map[string]int{"fish-carp": 2}, RewardFlow: 120, RewardItem: "hi-potion"},
	// ---- Hills: the Master Blacksmith
	{ID: "start-smith-wolves", Title: "Wolf Cull", Desc: "The Master Blacksmith wants 6 hills wolves culled before they reach the forge.",
		GiverNPCID: "master-blacksmith", RequireKills: map[string]int{"hills-wolf": 6}, RewardFlow: 400, RewardItem: "sabre"},
	{ID: "start-smith-shells", Title: "Beetle Plate", Desc: "Bring the Blacksmith 5 Beetle Shells for the plate-mail experiments.",
		GiverNPCID: "master-blacksmith", RequireItems: map[string]int{"beetle-shell": 5}, RewardFlow: 350, RewardItem: "broadsword"},
	{ID: "start-smith-hides", Title: "Rabbit Hides", Desc: "Bring 4 Rabbit Hides to line a new hauberk.",
		GiverNPCID: "master-blacksmith", RequireItems: map[string]int{"rabbit-hide": 4}, RewardFlow: 200, RewardItem: "wolf-hide-vest"},
	// ---- Caves: the Cartographer
	{ID: "start-cave-wings", Title: "Bat Wings", Desc: "Dunn the Cartographer needs 6 Bat Wings to waterproof his maps.",
		GiverNPCID: "cave-cartographer-dunn", RequireItems: map[string]int{"bat-wing": 6}, RewardFlow: 300, RewardItem: "beestinger"},
	{ID: "start-cave-skeletons", Title: "Skeleton Sweep", Desc: "Five skeletons are standing on Dunn's best survey line. Clear them.",
		GiverNPCID: "cave-cartographer-dunn", RequireKills: map[string]int{"skeleton": 5}, RewardFlow: 450, RewardItem: "hand-axe"},
	{ID: "start-cave-silk", Title: "Spider Silk for Dunn", Desc: "Dunn wants 4 Spider Silk for map-cord.",
		GiverNPCID: "cave-cartographer-dunn", RequireItems: map[string]int{"spider-silk": 4}, RewardFlow: 380, RewardItem: "silk-habit"},
	// ---- Swamp: the Peat Cutter
	{ID: "start-swamp-peat", Title: "Peat for Winter", Desc: "Odo the Peat Cutter needs 6 Peat dug from the Bog Deposit (mine it).",
		GiverNPCID: "peat-cutter-odo", RequireItems: map[string]int{"peat": 6}, RewardFlow: 500, RewardItem: "composite-bow"},
	{ID: "start-swamp-slimes", Title: "Slime Trouble", Desc: "Slimes keep eating Odo's peat stacks. Destroy 5 of them.",
		GiverNPCID: "peat-cutter-odo", RequireKills: map[string]int{"slime": 5}, RewardFlow: 550, RewardItem: "swift-blade"},
	{ID: "start-swamp-lizards", Title: "Lizard Tails", Desc: "Odo pays well for 4 Lizard Tails.",
		GiverNPCID: "peat-cutter-odo", RequireItems: map[string]int{"lizard-tail": 4}, RewardFlow: 480, RewardItem: "lizard-bone-greaves"},
	// ---- Town: the Mayor
	{ID: "start-town-crystal", Title: "Crystal Delivery", Desc: "Mayor Pell needs 2 Fire Crystals for the town's lamps.",
		GiverNPCID: "mayor-pell", RequireItems: map[string]int{"fire-crystal": 2}, RewardFlow: 200, RewardItem: "echo-drop"},
	{ID: "start-town-ingots", Title: "Your First Ingots", Desc: "Show the Mayor you can smelt: bring 2 Copper Ingots.",
		GiverNPCID: "mayor-pell", RequireItems: map[string]int{"copper-ingot": 2}, RewardFlow: 300, RewardItem: "copper-shortsword"},
	{ID: "start-town-ink", Title: "The Pigeon Signs", Desc: "Somebody keeps chalking PLEASE DO NOT DOCUMENT THE PIGEONS on the well. Bring the Mayor 2 vials of Ward Ink to make it official.",
		GiverNPCID: "mayor-pell", RequireItems: map[string]int{"ward-ink": 2}, RewardFlow: 600, RewardItem: "mini-bike-jacket"},

	// ---- bounties on the 24-hour mobs (the reward is a guaranteed item from the database, on top of
	// whatever the creature drops)
	{ID: "bounty-producer", Title: "Bounty: The Producer", Desc: "The Jiangshi Warden wants The Producer's voice silenced.",
		GiverNPCID: "jiangshi-warden", RequireKills: map[string]int{"The Producer (Voice Only)": 1}, RewardFlow: 40000, RewardItem: "operator's-tunic"},
	{ID: "bounty-custodian", Title: "Bounty: The Custodian Process", Desc: "The Broadcast Operator wants the Custodian Process stopped.",
		GiverNPCID: "broadcast-operator", RequireKills: map[string]int{"The Custodian Process": 1}, RewardFlow: 50000, RewardItem: "enigma's-curse"},
	{ID: "bounty-tesla", Title: "Bounty: Nikola Tesla", Desc: "The Warehouse Contact wants Tesla's Unfinished Tower finished -- by felling it.",
		GiverNPCID: "warehouse-contact", RequireKills: map[string]int{"Nikola Tesla, the Unfinished Tower": 1}, RewardFlow: 45000, RewardItem: "boots-of-winter"},
	{ID: "bounty-nidhogg", Title: "Bounty: Nidhogg", Desc: "The Eastwind Archivist wants the Root-Gnawer's gnawing stopped.",
		GiverNPCID: "eastwind-archivist", RequireKills: map[string]int{"Nidhogg, Root-Gnawer": 1}, RewardFlow: 60000, RewardItem: "knight's-plate"},
	{ID: "bounty-sleeper", Title: "Bounty: The Sleeper", Desc: "The Scar Keeper wants the Sleeper under the Sleeper College put back to sleep -- permanently.",
		GiverNPCID: "scar-keeper", RequireKills: map[string]int{"The Sleeper Beneath the Sleeper College": 1}, RewardFlow: 50000, RewardItem: "white-gloves"},
	{ID: "bounty-foreclosure", Title: "Bounty: No Foreclosure", Desc: "The Heikegani Dock Boss wants the crustocrat who would not foreclose foreclosed upon.",
		GiverNPCID: "heikegani-dock-boss", RequireKills: map[string]int{"No Foreclosure, the Crustocrat": 1}, RewardFlow: 45000, RewardItem: "blue-garter"},
	{ID: "bounty-arachne", Title: "Bounty: Arachne", Desc: "The Signal Runner wants the Queen of the Kuroshio Web cut down.",
		GiverNPCID: "kuroshio-signal-runner", RequireKills: map[string]int{"Arachne, Queen of the Kuroshio Web": 1}, RewardFlow: 42000, RewardItem: "utzu-mah-turi"},
	{ID: "bounty-noname", Title: "Bounty: No-Name", Desc: "The Shrine Tender wants the nine-tailed seat at Bacon's Table emptied.",
		GiverNPCID: "yokai-shrine-tender", RequireKills: map[string]int{"No-Name, the Kitsune of Nine Tails": 1}, RewardFlow: 70000, RewardItem: "loki's-favor"},
}
