package weather

import (
	"hash/fnv"
	"strings"
	"time"
)

// Elemental weather and the day-element cycle (FFXI-style).
//
// Elemental weather is per zone and stateless: the weather in a zone is a pure function of the zone
// ID and the current 20-minute slot, so it is stable across server restarts and identical for every
// player standing in that zone. Each zone has a weighted table biased toward the elements that suit
// its terrain. The day cycle is a fixed 8-day rotation where one "day" lasts one real hour.
//
// Both feed crafting (server/craft): a crystal whose element matches the day and/or the weather
// synthesizes better; one opposed by them is more likely to be lost.

// Element is one of the eight FFXI elements (or "" for none).
type Element string

const (
	None      Element = ""
	Fire      Element = "fire"
	Earth     Element = "earth"
	Water     Element = "water"
	Wind      Element = "wind"
	Ice       Element = "ice"
	Lightning Element = "lightning"
	Light     Element = "light"
	Dark      Element = "dark"
)

// AllElements in FFXI day order (Firesday ... Darksday).
var AllElements = []Element{Fire, Earth, Water, Wind, Ice, Lightning, Light, Dark}

// DayLength is how long one elemental day lasts in real time.
const DayLength = time.Hour

// SlotLength is how long a zone keeps one weather before it can change.
const SlotLength = 20 * time.Minute

// DayName returns the FFXI-style day name, e.g. "Firesday".
func DayName(e Element) string {
	switch e {
	case Fire:
		return "Firesday"
	case Earth:
		return "Earthsday"
	case Water:
		return "Watersday"
	case Wind:
		return "Windsday"
	case Ice:
		return "Iceday"
	case Lightning:
		return "Lightningday"
	case Light:
		return "Lightsday"
	case Dark:
		return "Darksday"
	}
	return "Unknownday"
}

// DayElement is the element of the elemental day containing t (UTC).
func DayElement(t time.Time) Element {
	idx := int((t.UTC().Unix() / int64(DayLength.Seconds())) % int64(len(AllElements)))
	return AllElements[idx]
}

// NextDayChange is when the current elemental day ends.
func NextDayChange(t time.Time) time.Time {
	d := int64(DayLength.Seconds())
	return time.Unix((t.UTC().Unix()/d+1)*d, 0).UTC()
}

// Sky is the weather in a zone: a display name, the element it carries and its strength
// (1 = single, 2 = double, as in "Heat Wave" vs "Hot Spell"; 0 = no element).
type Sky struct {
	Name    string
	Element Element
	Power   int
}

var skies = map[Element][2]string{
	Fire:      {"Hot Spell", "Heat Wave"},
	Earth:     {"Dust Storm", "Sand Storm"},
	Water:     {"Rain", "Squalls"},
	Wind:      {"Windy", "Gales"},
	Ice:       {"Snow", "Blizzards"},
	Lightning: {"Thunder", "Thunderstorms"},
	Light:     {"Auroras", "Stellar Glare"},
	Dark:      {"Gloom", "Darkness"},
}

type skyWeight struct {
	e Element
	w int
}

// zoneBias lists the elements a zone leans toward; weights are relative. "Clear" (no element) is
// always possible. Zones not listed use defaultBias.
var zoneBias = map[int][]skyWeight{
	0:   {{Wind, 3}, {Light, 3}, {Earth, 2}, {Water, 1}},     // Meadow
	1:   {{Wind, 4}, {Earth, 3}, {Ice, 2}, {Lightning, 1}},   // Hills
	2:   {{Earth, 4}, {Dark, 3}, {Ice, 1}},                   // Caves
	3:   {{Water, 4}, {Dark, 3}, {Earth, 1}},                 // Swampville
	4:   {{Light, 3}, {Fire, 2}, {Wind, 2}},                  // New Handington
	10:  {{Light, 3}, {Wind, 3}, {Earth, 2}},                 // Fledgling Field
	11:  {{Wind, 4}, {Earth, 2}, {Light, 2}},                 // Wren's Perch
	12:  {{Dark, 4}, {Earth, 3}},                             // Roost Deeps
	13:  {{Water, 4}, {Dark, 2}, {Earth, 2}},                 // Pellet Marsh
	14:  {{Lightning, 3}, {Wind, 3}, {Light, 3}},             // Observation Tor
	15:  {{Ice, 4}, {Wind, 4}, {Lightning, 1}},               // Eastwind Ridge
	16:  {{Dark, 2}, {Light, 2}, {Earth, 2}},                 // Cataloguer's Stacks
	17:  {{Water, 3}, {Dark, 4}, {Ice, 2}},                   // Footnote Moor
	18:  {{Dark, 3}, {Ice, 3}, {Lightning, 2}},               // Protocol Wall
	19:  {{Wind, 3}, {Water, 3}, {Light, 1}},                 // Handington Docks
	20:  {{Water, 3}, {Wind, 3}, {Light, 2}},                 // Open Balance Quay
	21:  {{Fire, 4}, {Earth, 3}, {Light, 2}},                 // Salt Flats
	22:  {{Light, 3}, {Wind, 2}, {Water, 2}},                 // Pilgrim's Terraces
	23:  {{Water, 5}, {Wind, 2}, {Dark, 1}},                  // Tidepool Fields
	24:  {{Water, 4}, {Dark, 2}, {Earth, 2}},                 // Cormorant Marsh
	25:  {{Lightning, 3}, {Wind, 3}, {Light, 2}, {Dark, 1}},  // Shrine Stair
	26:  {{Water, 4}, {Dark, 3}},                             // Sunken Chapel Coast
	28:  {{Light, 3}, {Wind, 2}, {Earth, 2}},                 // Tanuki Council Road
	29:  {{Wind, 5}, {Ice, 2}, {Lightning, 2}},               // Web Cliffs
	30:  {{Dark, 3}, {Ice, 3}, {Water, 2}},                   // Foreclosure Anchorage
	31:  {{Light, 3}, {Dark, 3}, {Lightning, 2}, {Water, 2}}, // The Wellspring of Tides
	32:  {{Water, 3}, {Wind, 3}},                             // Aboard the Open Balance
	200: {{Fire, 2}, {Lightning, 2}, {Wind, 2}},              // Detroit apartment district
	205: {{Water, 4}, {Dark, 2}},                             // Osaka Underport
	206: {{Water, 3}, {Wind, 3}, {Dark, 2}},                  // Kuroshio Coast
}

var defaultBias = []skyWeight{{Wind, 2}, {Earth, 2}, {Water, 2}, {Fire, 1}, {Ice, 1}, {Light, 1}, {Dark, 1}, {Lightning, 1}}

// clearWeight is the weight of "no elemental weather" (a plain Clear sky).
const clearWeight = 6

// ZoneSky returns the weather in zoneID at time t.
func ZoneSky(zoneID int, t time.Time) Sky {
	slot := t.UTC().Unix() / int64(SlotLength.Seconds())
	h := fnv.New64a()
	var buf [16]byte
	for i := 0; i < 8; i++ {
		buf[i] = byte(uint64(zoneID) >> (8 * i))
		buf[8+i] = byte(uint64(slot) >> (8 * i))
	}
	_, _ = h.Write(buf[:])
	r := h.Sum64()

	bias, ok := zoneBias[zoneID]
	if !ok {
		bias = defaultBias
	}
	total := clearWeight
	for _, b := range bias {
		total += b.w
	}
	pick := int(r % uint64(total))
	if pick < clearWeight {
		return Sky{Name: "Clear"}
	}
	pick -= clearWeight
	for _, b := range bias {
		if pick < b.w {
			// one in four elemental skies is the strong (double) variety
			power := 1
			if (r>>32)%4 == 0 {
				power = 2
			}
			return Sky{Name: skies[b.e][power-1], Element: b.e, Power: power}
		}
		pick -= b.w
	}
	return Sky{Name: "Clear"}
}

// Opposed returns the element that overcomes e (a crystal of element e is hurt on that element's
// day or under its weather): Fire<-Water<-Lightning<-Earth<-Wind<-Ice<-Fire, Light<->Dark.
func Opposed(e Element) Element {
	switch e {
	case Fire:
		return Water
	case Water:
		return Lightning
	case Lightning:
		return Earth
	case Earth:
		return Wind
	case Wind:
		return Ice
	case Ice:
		return Fire
	case Light:
		return Dark
	case Dark:
		return Light
	}
	return None
}

// ParseElement maps a crystal/element word ("fire", "Fire Crystal") to an Element.
func ParseElement(s string) Element {
	s = strings.ToLower(s)
	for _, e := range AllElements {
		if strings.Contains(s, string(e)) {
			return e
		}
	}
	if strings.Contains(s, "thunder") {
		return Lightning
	}
	return None
}
