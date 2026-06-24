// Package autotranslate implements FFXI-style auto-translate phrase selection.
//
// Players type [phrase-id] or [jp:phrase-id] to embed a pre-built phrase that
// renders in both English and Japanese. Phrases are bracket-wrapped in output:
//   [Hello!] / [こんにちは！]
//
// VS0 ships a core phrase set covering combat, party, travel, and social interactions.
// The phrase ID system is numeric (matching FFXI's packed-byte encoding) + a human
// alias for MUD convenience: [at:greet] or [100].
package autotranslate

import "strings"

// Lang selects the display language for a phrase.
type Lang int

const (
	EN   Lang = iota
	JP
	BOTH // receives phrases in both EN and JP side-by-side
)

// ParseLang converts "en", "jp", or "both" to the corresponding Lang constant.
// Returns EN and false if unrecognized.
func ParseLang(s string) (Lang, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "en":
		return EN, true
	case "jp":
		return JP, true
	case "both":
		return BOTH, true
	}
	return EN, false
}

// LangName returns the canonical lowercase name for a Lang constant.
func LangName(l Lang) string {
	switch l {
	case JP:
		return "jp"
	case BOTH:
		return "both"
	default:
		return "en"
	}
}

// Phrase is a single auto-translate entry.
type Phrase struct {
	ID      int
	Alias   string // shorthand for MUD input, e.g. "greet"
	EN      string
	JP      string
	Category string
}

// Render returns the phrase rendered for the given language, bracket-wrapped.
// BOTH renders EN and JP side-by-side.
func (p Phrase) Render(l Lang) string {
	switch l {
	case JP:
		return "[" + p.JP + "]"
	case BOTH:
		return p.RenderBoth()
	default:
		return "[" + p.EN + "]"
	}
}

// RenderBoth returns EN/JP side-by-side as FFXI would display to a bilingual party.
func (p Phrase) RenderBoth() string {
	return "[" + p.EN + "] [" + p.JP + "]"
}

// ── Phrase database ────────────────────────────────────────────────────────────

// Phrases is the VS0 auto-translate dictionary.
// ID ranges mirror FFXI's encoding: 100-199 greetings, 200-299 party, 300-399 combat,
// 400-499 status, 500-599 requests, 600-699 locations, 700-799 emotions.
var Phrases = []Phrase{
	// Greetings
	{100, "greet",    "Hello!",               "こんにちは！",         "Greetings"},
	{101, "bye",      "Goodbye!",              "さようなら！",         "Greetings"},
	{102, "ty",       "Thank you!",            "ありがとう！",         "Greetings"},
	{103, "yw",       "You're welcome!",       "どういたしまして！",   "Greetings"},
	{104, "sorry",    "I'm sorry.",            "ごめんなさい。",       "Greetings"},
	{105, "np",       "No problem.",           "問題ないよ。",         "Greetings"},
	{106, "wb",       "Welcome back!",         "おかえり！",           "Greetings"},
	{107, "gm",       "Good morning!",         "おはよう！",           "Greetings"},
	{108, "gn",       "Good night!",           "おやすみ！",           "Greetings"},
	{109, "gl",       "Good luck!",            "頑張って！",           "Greetings"},

	// Party
	{200, "invite",   "Join my party?",        "パーティーに入る？",    "Party"},
	{201, "disband",  "Disbanding party.",     "パーティー解散します。","Party"},
	{202, "wait",     "Wait a moment.",        "ちょっと待って。",      "Party"},
	{203, "ready",    "I'm ready.",            "準備完了。",            "Party"},
	{204, "notready", "Not ready yet.",        "まだ準備できてない。",  "Party"},
	{205, "follow",   "Follow me.",            "ついてきて。",          "Party"},
	{206, "stay",     "Stay here.",            "ここにいて。",          "Party"},
	{207, "atk",      "Attack!",               "攻撃して！",            "Party"},
	{208, "tank",     "I'll tank.",            "私がタンクします。",    "Party"},
	{209, "heal",     "I'll heal.",            "私がヒールします。",    "Party"},
	{210, "dd",       "I'll deal damage.",     "DDやります。",          "Party"},
	{211, "pull",     "I'll pull.",            "私がプルします。",      "Party"},
	{212, "camp",     "Set up camp here.",     "ここでキャンプ。",      "Party"},
	{213, "break",    "Taking a short break.", "少し休憩。",            "Party"},
	{214, "lost",     "I'm lost.",             "迷子になった。",        "Party"},

	// Combat
	{300, "links",    "It's linking!",         "リンクしてる！",        "Combat"},
	{301, "claim",    "I'll claim it.",        "私がクレイムします。",  "Combat"},
	{302, "flee",     "Run away!",             "逃げて！",              "Combat"},
	{303, "aggro",    "Aggro!",                "アグロった！",          "Combat"},
	{304, "sleep",    "Sleeping target.",      "スリープします。",      "Combat"},
	{305, "stun",     "Stunning target.",      "スタンします。",        "Combat"},
	{306, "bind",     "Binding target.",       "バインドします。",      "Combat"},
	{307, "nuke",     "Nuking now.",           "魔法打ちます。",        "Combat"},
	{308, "ws",       "Weapon skill!",         "WSします！",            "Combat"},
	{309, "sc",       "Skillchain!",           "連携します！",          "Combat"},
	{310, "mb",       "Magic burst!",          "マジックバースト！",    "Combat"},
	{311, "lowmp",    "Running low on MP.",    "MPが少ない。",          "Combat"},
	{312, "lowhe",    "Low HP!",               "HPがやばい！",          "Combat"},

	// Status
	{400, "silenced",  "I'm silenced.",        "沈黙状態です。",        "Status"},
	{401, "blind",     "I'm blinded.",         "暗闇状態です。",        "Status"},
	{402, "poison",    "I'm poisoned.",        "毒状態です。",          "Status"},
	{403, "paralyzed", "I'm paralyzed.",       "麻痺状態です。",        "Status"},
	{404, "slow",      "I'm slowed.",          "スロー状態です。",      "Status"},
	{405, "weakened",  "I'm weakened.",        "衰弱状態です。",        "Status"},
	{406, "resting",   "Resting.",             "休憩中。",              "Status"},
	{407, "medding",   "Medding.",             "瞑想中。",              "Status"},
	{408, "afk",       "Be right back.",       "少し離席します。",      "Status"},
	{409, "crafting",  "Crafting.",            "製作中。",              "Status"},
	{410, "synthing",  "Synthesizing.",        "シンセ中。",            "Status"},

	// Requests
	{500, "cure",     "Please cure me.",       "ケアルお願い。",        "Requests"},
	{501, "raise",    "Please raise me.",      "蘇生お願い。",          "Requests"},
	{502, "raise2",   "Please Raise II.",      "レイズIIお願い。",      "Requests"},
	{503, "protect",  "Please cast Protect.",  "プロテクお願い。",      "Requests"},
	{504, "shell",    "Please cast Shell.",    "シェルお願い。",        "Requests"},
	{505, "haste",    "Please cast Haste.",    "ヘイストお願い。",      "Requests"},
	{506, "refresh",  "Please cast Refresh.",  "リフレお願い。",        "Requests"},
	{507, "regen",    "Please cast Regen.",    "リジェネお願い。",      "Requests"},
	{508, "sneak",    "Please cast Sneak.",    "スニークお願い。",      "Requests"},
	{509, "invis",    "Please cast Invisible.","インビジお願い。",      "Requests"},

	// Locations (TRAPX districts)
	{600, "apt",      "Detroit Apartment",     "デトロイトアパート",    "Locations"},
	{601, "school",   "Detroit School",        "デトロイト学校",        "Locations"},
	{602, "conveni",  "Osaka Convenience Store","大阪コンビニ",         "Locations"},
	{603, "archive",  "Cairngorms Archive",    "ケアンゴームアーカイブ","Locations"},
	{604, "vatican",  "Vatican Corridors",     "バチカン回廊",          "Locations"},
	{605, "underport","Osaka Underport",       "大阪地下港",            "Locations"},
	{606, "coast",    "Kuroshio Coast",        "黒潮海岸",              "Locations"},
	{607, "table",    "Bacon's Table",         "ベーコンのテーブル",    "Locations"},
	{608, "freq",     "The Frequency HQ",      "フリークエンシー本部",  "Locations"},
	{609, "bloc",     "The Bloc HQ",           "ブロック本部",          "Locations"},
	{610, "procure",  "Procurement Houses",    "調達ハウス",            "Locations"},

	// Emotions / social
	{700, "lol",      "Ha ha ha!",             "笑笑笑！",              "Emotions"},
	{701, "sigh",     "*sigh*",                "はあ……",               "Emotions"},
	{702, "nice",     "Nice one!",             "ナイス！",              "Emotions"},
	{703, "gg",       "Good game.",            "お疲れ様。",            "Emotions"},
	{704, "wow",      "Amazing!",              "すごい！",              "Emotions"},
	{705, "nooo",     "Nooooo!!!",             "なんで！！！",          "Emotions"},
	{706, "afaik",    "I think so.",           "そうだと思う。",        "Emotions"},
	{707, "idk",      "I don't know.",         "わからない。",          "Emotions"},

	// Channel 11 / TRAPX flavor
	{800, "ch11",     "Are you watching Channel 11?", "チャンネル11見てる？", "TRAPX"},
	{801, "cast",     "CAST terminal message", "CASTターミナルより",    "TRAPX"},
	{802, "fo",       "Field Office secure",   "フィールドオフィス確保","TRAPX"},
	{803, "bikegang", "Mini bike crew here",   "ミニバイク軍団参上",    "TRAPX"},
	{804, "dragon",   "The Dragon appeared!",  "ドラゴンが現れた！",    "TRAPX"},
	{805, "dispatch", "Emily dispatch received","エミリー指令受信",     "TRAPX"},
}

// ── Lookup ─────────────────────────────────────────────────────────────────────

// byID is built once at init for O(1) ID lookup.
var byID map[int]*Phrase

// byAlias is built once at init for O(1) alias lookup.
var byAlias map[string]*Phrase

func init() {
	byID = make(map[int]*Phrase, len(Phrases))
	byAlias = make(map[string]*Phrase, len(Phrases))
	for i := range Phrases {
		p := &Phrases[i]
		byID[p.ID] = p
		if p.Alias != "" {
			byAlias[strings.ToLower(p.Alias)] = p
		}
	}
}

// Lookup finds a phrase by numeric ID or alias string.
// Returns nil if not found.
func Lookup(key string) *Phrase {
	key = strings.TrimSpace(strings.ToLower(key))
	if p, ok := byAlias[key]; ok {
		return p
	}
	// Try numeric.
	n := 0
	for _, c := range key {
		if c < '0' || c > '9' {
			return nil
		}
		n = n*10 + int(c-'0')
	}
	return byID[n]
}

// ByCategory returns all phrases in the named category (case-insensitive).
func ByCategory(cat string) []Phrase {
	cat = strings.ToLower(cat)
	var out []Phrase
	for _, p := range Phrases {
		if strings.ToLower(p.Category) == cat {
			out = append(out, p)
		}
	}
	return out
}

// Categories returns the unique category names in definition order.
func Categories() []string {
	seen := make(map[string]bool)
	var out []string
	for _, p := range Phrases {
		if !seen[p.Category] {
			seen[p.Category] = true
			out = append(out, p.Category)
		}
	}
	return out
}

// ExpandLine replaces all [phrase-key] tokens in a chat line with their rendered form.
// lang selects the display language for the current player's client.
// Input: "meeting at [apt] [ready]"
// Output (EN): "meeting at [Detroit Apartment] [I'm ready.]"
func ExpandLine(line string, lang Lang) string {
	var out strings.Builder
	i := 0
	for i < len(line) {
		if line[i] != '[' {
			out.WriteByte(line[i])
			i++
			continue
		}
		j := strings.IndexByte(line[i+1:], ']')
		if j < 0 {
			out.WriteByte(line[i])
			i++
			continue
		}
		token := line[i+1 : i+1+j]
		if p := Lookup(token); p != nil {
			out.WriteString(p.Render(lang))
		} else {
			out.WriteByte('[')
			out.WriteString(token)
			out.WriteByte(']')
		}
		i = i + 1 + j + 1
	}
	return out.String()
}
