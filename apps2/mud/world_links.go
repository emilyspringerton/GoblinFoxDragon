package main

// world_links.go -- ties the New Handington town (zone 4) into the walkable MUD world. It used to be
// reachable only from the GUI/telecrystal, so a telnet player could never get to its vendors.

func init() {
	exits[0]["up"] = 4 // Meadow -> the stairway up to New Handington
	exits[4] = map[string]int{"down": 0}
	zoneDesc[4] = "[EASTWIND: NEW HANDINGTON / Free Town / Market] A hillside town of whitewashed stalls and clay-tile roofs, reached by a switchback stair above the Meadow. " +
		"A hand-drawn map is pinned to the well. The crystal seller's awning hums faintly. Somebody has chalked a small, polite sign: PLEASE DO NOT DOCUMENT THE PIGEONS."
}
