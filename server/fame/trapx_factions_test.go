package fame

import "testing"

func TestTRAPXFactionNameFrequency(t *testing.T) {
	if TRAPXFactionName(Sandoria) != "The Frequency" {
		t.Errorf("expected The Frequency, got %q", TRAPXFactionName(Sandoria))
	}
}

func TestTRAPXFactionNameBloc(t *testing.T) {
	if TRAPXFactionName(Bastok) != "The Bloc" {
		t.Errorf("expected The Bloc, got %q", TRAPXFactionName(Bastok))
	}
}

func TestTRAPXFactionNameProcurement(t *testing.T) {
	if TRAPXFactionName(Windurst) != "Procurement Houses" {
		t.Errorf("expected Procurement Houses, got %q", TRAPXFactionName(Windurst))
	}
}

func TestTRAPXFactionNameNeutral(t *testing.T) {
	if TRAPXFactionName(Neutral) != "Unaligned" {
		t.Errorf("expected Unaligned, got %q", TRAPXFactionName(Neutral))
	}
}

func TestTRAPXFactionDescFrequency(t *testing.T) {
	d := TRAPXFactionDesc(Sandoria)
	if d == "" {
		t.Error("expected non-empty description for Sandoria/Frequency")
	}
}

func TestTRAPXFactionBenefitRank5Frequency(t *testing.T) {
	b := TRAPXFactionBenefit(Sandoria, 5)
	if b == "" {
		t.Error("expected rank 5 benefit for Frequency")
	}
}

func TestTRAPXFactionBenefitRank3Bloc(t *testing.T) {
	b := TRAPXFactionBenefit(Bastok, 3)
	if b == "" {
		t.Error("expected rank 3 benefit for Bloc")
	}
}

func TestTRAPXFactionBenefitRank1Procurement(t *testing.T) {
	b := TRAPXFactionBenefit(Windurst, 1)
	if b == "" {
		t.Error("expected default benefit for Procurement at rank 1")
	}
}

func TestTRAPXFactionBenefitNeutral(t *testing.T) {
	b := TRAPXFactionBenefit(Neutral, 5)
	if b == "" {
		t.Error("expected fallback benefit string for Neutral")
	}
}

func TestTRAPXFactions(t *testing.T) {
	factions := TRAPXFactions()
	if len(factions) != 3 {
		t.Errorf("expected 3 TRAPX factions, got %d", len(factions))
	}
}

func TestTRAPXFactionNamesNotEqualNationNames(t *testing.T) {
	// Ensure the overlay doesn't accidentally match the fantasy names.
	if TRAPXFactionName(Sandoria) == NationName(Sandoria) {
		t.Error("TRAPX faction name should not equal fantasy nation name")
	}
}
