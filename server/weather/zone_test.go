package weather

import (
	"testing"
	"time"
)

func TestDayCycleRotatesThroughAllEightElements(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	seen := map[Element]bool{}
	for h := 0; h < 8; h++ {
		seen[DayElement(base.Add(time.Duration(h)*time.Hour+time.Minute))] = true
	}
	if len(seen) != 8 {
		t.Fatalf("expected all 8 elemental days within 8 hours, got %d", len(seen))
	}
	if DayElement(base) != DayElement(base.Add(59*time.Minute)) {
		t.Errorf("a day should last a full hour")
	}
	if DayElement(base) != DayElement(base.Add(8*time.Hour)) {
		t.Errorf("the cycle should repeat every 8 hours")
	}
	if got := NextDayChange(base.Add(10 * time.Minute)); !got.Equal(base.Add(time.Hour)) {
		t.Errorf("NextDayChange = %v", got)
	}
}

func TestZoneSkyIsStableAndZoneSpecific(t *testing.T) {
	now := time.Date(2026, 5, 5, 12, 7, 0, 0, time.UTC)
	if ZoneSky(3, now) != ZoneSky(3, now.Add(5*time.Minute)) {
		t.Errorf("weather must not change within a slot")
	}
	varied := map[Sky]bool{}
	for i := 0; i < 300; i++ {
		varied[ZoneSky(3, now.Add(time.Duration(i)*SlotLength))] = true
	}
	if len(varied) < 4 {
		t.Errorf("a zone should see a variety of weather over time, got %d kinds", len(varied))
	}
}

func TestSwampLeansWaterAndDark(t *testing.T) {
	now := time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)
	count := map[Element]int{}
	for i := 0; i < 3000; i++ {
		s := ZoneSky(3, now.Add(time.Duration(i)*SlotLength))
		count[s.Element]++
	}
	if count[Water]+count[Dark] <= count[Fire]+count[Ice]+count[Lightning]+count[Light]+count[Wind] {
		t.Errorf("swamp should favour water/dark weather: %v", count)
	}
	if count[None] == 0 {
		t.Errorf("clear skies should occur")
	}
}

func TestDoubleWeatherUsesStrongNames(t *testing.T) {
	now := time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)
	var saw1, saw2 bool
	for i := 0; i < 4000; i++ {
		s := ZoneSky(15, now.Add(time.Duration(i)*SlotLength))
		if s.Element == Ice && s.Power == 1 && s.Name == "Snow" {
			saw1 = true
		}
		if s.Element == Ice && s.Power == 2 && s.Name == "Blizzards" {
			saw2 = true
		}
	}
	if !saw1 || !saw2 {
		t.Errorf("expected both Snow and Blizzards on the ridge (snow=%v blizzards=%v)", saw1, saw2)
	}
}

func TestOpposedElementsAreAClosedChain(t *testing.T) {
	e := Fire
	for i := 0; i < 6; i++ {
		e = Opposed(e)
	}
	if e != Fire {
		t.Errorf("Fire chain should loop in 6 steps, ended on %v", e)
	}
	if Opposed(Light) != Dark || Opposed(Dark) != Light {
		t.Errorf("light and dark oppose each other")
	}
	if ParseElement("Thunder Crystal") != Lightning || ParseElement("earth-crystal") != Earth {
		t.Errorf("ParseElement mapping")
	}
}
