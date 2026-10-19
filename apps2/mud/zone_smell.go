package main

// zoneSmell is the one-line olfactory read every zone gives on entry (cmdGo -> cmdLook) and on
// `look`. Voice follows TYLER's house semantics: deadpan, documentary-flat, no explanation
// (Emily OS never explains herself -- the room just tightens), faction tells carried by the
// smell itself, frequencies named only where the lore already names them (Stolas 7.83 Hz,
// Vassago 11.11 Hz, Amon 1.618 Hz), and a couple of sentences that stop before they finish
// (Tyler never completes a self-defining sentence).
var zoneSmell = map[int]string{
	0: "Cut grass, wet earth, and under it something faintly sweet -- worm castings, probably.",
	1: "Wind-scoured heather and cold stone. Nothing up here has been cooked, burned, or filed.",
	2: "Wet limestone, bat guano, and old iron. The cave smells the way a locked drawer smells.",
	3: "Rot-sweet mangrove, standing water, and a green, metallic edge you taste before you smell.",
	4: "Woodsmoke, hot bread, and boot leather. New Handington smells like somewhere that expects you to stay.",

	200: "Fryer oil, wet laundry, and mini-bike exhaust. Under it, the hot-dust smell of a CRT that has been on for years. Someone upstairs is being recorded.",
	201: "Floor wax that stopped being applied a decade ago, chalk dust, and ozone from the lab terminal that never shut off. Emily OS notes your arrival. The room tightens.",
	202: "Microwaved rice, instant coffee, cold rain on a vinyl awning. The counter smells like it has been wiped by three different factions and trusted by none.",
	203: "Dry paper, cold air, and a faint resin note from the box seals. Nothing here has decayed on purpose. Somewhere an owl has already catalogued you.",
	204: "Salt water in stone that has never been near a sea. Brine, wet shell, and candle smoke that is not the Church's. The smell is older than the corridor.",
	205: "Bilge water, diesel, and crab shell drying on hot pipe. Sweet and rotten at once. A paying smell -- somebody here is owed something.",
	206: "Kelp rot, cold iron, and the clean, empty smell of a tide that came in and left nothing. The buoys give off no scent at all, which is the scent.",
	207: "Incense over mildew, old pallet wood, a rice-wine sweetness. Every seven minutes it smells like a different host. You were not told which one it is now.",

	// GFD-993944 dungeons (dungeonZoneBase + mob.DungeonRoster index, in order).
	dungeonZoneBase + 0: "Sealed paper and held breath. Dust that has been told to stay where it is.",
	dungeonZoneBase + 1: "Hot copper and a clean, electrical hum you can smell more than hear. Seven point eight three, steady, and you are standing in it.",
	dungeonZoneBase + 2: "Cold ash, tallow, and the sourness of a hall that kept its feast for people who never came back to it.",
	dungeonZoneBase + 3: "Pine resin, hot tea, and stone warmed by a long afternoon on the far side of something. The wall smells like it is listening.",
	dungeonZoneBase + 4: "Peat smoke, wet wool, and rain that has not decided yet. Something here has been mourning for a long time, and the smell is how it does it.",
	dungeonZoneBase + 5: "Beeswax, spilled wine, and a faint animal warmth. Somebody set the table. Somebody is about to say who it was for and does not",
	dungeonZoneBase + 6: "No smell. That is the smell: air that has never been breathed, a wing no one filed a reason to open.",
	dungeonZoneBase + 7: "Sweat, sand, and sharpened steel. It smells like a place that has only ever asked one question.",
}
