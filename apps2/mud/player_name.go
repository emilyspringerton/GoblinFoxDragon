package main

// player_name.go — real name-collision gap, founder real-time, 2026-09-12: "it should not allow
// the guest to login as EMILY thats my character on the ssh also Emily should be taken too we
// should do normalization so that we can have names in urls if we really wanted to."
//
// normalizePlayerName is the one, explicit, named place this codebase decides what "the same
// name" means -- a real, deliberate concept, not just an ad-hoc strings.EqualFold sprinkled at
// each comparison site. Two real properties fall out of it together: (1) case-insensitive
// comparison ("EMILY" and "Emily" are the same name, closing the exact gap reported above), and
// (2) the character set every accepted name is already restricted to (validateSSHClaimName,
// ssh_listener.go -- letters and digits only, 2-20 characters, reused for guest names too as of
// this same fix) is already URL-safe by construction, so a future feature that puts a player's
// name into a URL path (a public profile page, a WOTAN leaderboard link, etc.) never needs a
// separate escaping step -- lower-casing here is the only real transformation a URL slug would
// ever need on top of what's already guaranteed. Not wired into any URL-producing code today --
// named and built now specifically so it doesn't need retrofitting later, per the founder's own
// "if we really wanted to" framing.
func normalizePlayerName(name string) string {
	return toLowerASCII(name)
}

// nameCollidesWithOnlinePlayer reports whether any player currently in players (guest or
// SSH-bound alike) already has the given name, compared via normalizePlayerName -- callers
// (handleConn's own guest-name-entry path) hold gw.mu for the duration, matching every other
// direct read of the shared players map elsewhere in this codebase. A real, deliberate limit:
// this only ever sees who's online RIGHT NOW -- a permanent, offline SSH-bound character's name
// is checked separately via IDUNA's own GetCharacterByName, not here.
func nameCollidesWithOnlinePlayer(players map[string]*player, name string) bool {
	normalized := normalizePlayerName(name)
	for _, other := range players {
		if normalizePlayerName(other.name) == normalized {
			return true
		}
	}
	return false
}

// toLowerASCII is a small, explicit ASCII-only lowercase -- correct and sufficient here because
// validateSSHClaimName already restricts every real name to 'a'-'z'/'A'-'Z'/'0'-'9' before it can
// ever reach this function; strings.ToLower's own full-Unicode case-folding machinery would be
// real, unnecessary generality for an input space this narrow.
func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
