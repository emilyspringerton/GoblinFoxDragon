package autotranslate_test

import (
	"strings"
	"testing"

	"dragonsnshit/server/autotranslate"
)

func TestLookupByAlias(t *testing.T) {
	p := autotranslate.Lookup("greet")
	if p == nil {
		t.Fatal("expected phrase for alias 'greet'")
	}
	if p.EN != "Hello!" {
		t.Errorf("EN: want Hello!, got %q", p.EN)
	}
	if p.JP != "こんにちは！" {
		t.Errorf("JP: want こんにちは！, got %q", p.JP)
	}
}

func TestLookupByID(t *testing.T) {
	p := autotranslate.Lookup("100")
	if p == nil {
		t.Fatal("expected phrase for ID 100")
	}
	if p.Alias != "greet" {
		t.Errorf("alias: want greet, got %q", p.Alias)
	}
}

func TestLookupCaseInsensitive(t *testing.T) {
	if autotranslate.Lookup("GREET") == nil {
		t.Error("lookup should be case-insensitive")
	}
}

func TestLookupMissing(t *testing.T) {
	if autotranslate.Lookup("nonexistent") != nil {
		t.Error("expected nil for unknown key")
	}
	if autotranslate.Lookup("99999") != nil {
		t.Error("expected nil for unknown ID")
	}
}

func TestRenderEN(t *testing.T) {
	p := autotranslate.Lookup("ready")
	if p == nil {
		t.Fatal("no phrase for 'ready'")
	}
	got := p.Render(autotranslate.EN)
	if got != "[I'm ready.]" {
		t.Errorf("got %q", got)
	}
}

func TestRenderJP(t *testing.T) {
	p := autotranslate.Lookup("ready")
	got := p.Render(autotranslate.JP)
	if got != "[準備完了。]" {
		t.Errorf("got %q", got)
	}
}

func TestRenderBoth(t *testing.T) {
	p := autotranslate.Lookup("ty")
	got := p.RenderBoth()
	if !strings.Contains(got, "[Thank you!]") || !strings.Contains(got, "[ありがとう！]") {
		t.Errorf("got %q", got)
	}
}

func TestExpandLineEN(t *testing.T) {
	got := autotranslate.ExpandLine("meeting at [apt] [ready]", autotranslate.EN)
	want := "meeting at [Detroit Apartment] [I'm ready.]"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestExpandLineJP(t *testing.T) {
	got := autotranslate.ExpandLine("集合は [apt] [ready]", autotranslate.JP)
	if !strings.Contains(got, "[デトロイトアパート]") || !strings.Contains(got, "[準備完了。]") {
		t.Errorf("got %q", got)
	}
}

func TestExpandLineByID(t *testing.T) {
	got := autotranslate.ExpandLine("[200]", autotranslate.EN)
	if got != "[Join my party?]" {
		t.Errorf("got %q", got)
	}
}

func TestExpandLineUnknownPreserved(t *testing.T) {
	got := autotranslate.ExpandLine("[foobar]", autotranslate.EN)
	if got != "[foobar]" {
		t.Errorf("got %q", got)
	}
}

func TestExpandLineNoTokens(t *testing.T) {
	input := "just normal chat"
	got := autotranslate.ExpandLine(input, autotranslate.EN)
	if got != input {
		t.Errorf("got %q", got)
	}
}

func TestByCategory(t *testing.T) {
	combat := autotranslate.ByCategory("combat")
	if len(combat) == 0 {
		t.Error("expected combat phrases")
	}
	for _, p := range combat {
		if p.Category != "Combat" {
			t.Errorf("wrong category: %q", p.Category)
		}
	}
}

func TestCategories(t *testing.T) {
	cats := autotranslate.Categories()
	if len(cats) == 0 {
		t.Error("expected at least one category")
	}
	seen := make(map[string]bool)
	for _, c := range cats {
		if seen[c] {
			t.Errorf("duplicate category: %q", c)
		}
		seen[c] = true
	}
}

func TestTRAPXPhrases(t *testing.T) {
	cases := []struct{ alias, en string }{
		{"ch11", "Are you watching Channel 11?"},
		{"fo", "Field Office secure"},
		{"bikegang", "Mini bike crew here"},
		{"dragon", "The Dragon appeared!"},
	}
	for _, tc := range cases {
		p := autotranslate.Lookup(tc.alias)
		if p == nil {
			t.Errorf("no phrase for %q", tc.alias)
			continue
		}
		if p.EN != tc.en {
			t.Errorf("alias %q: want EN %q, got %q", tc.alias, tc.en, p.EN)
		}
	}
}

func TestAllPhrasesHaveEN(t *testing.T) {
	for _, p := range autotranslate.Phrases {
		if p.EN == "" {
			t.Errorf("phrase ID %d has no EN text", p.ID)
		}
		if p.JP == "" {
			t.Errorf("phrase ID %d has no JP text", p.ID)
		}
	}
}

func TestAllIDsUnique(t *testing.T) {
	seen := make(map[int]bool)
	for _, p := range autotranslate.Phrases {
		if seen[p.ID] {
			t.Errorf("duplicate ID: %d", p.ID)
		}
		seen[p.ID] = true
	}
}
