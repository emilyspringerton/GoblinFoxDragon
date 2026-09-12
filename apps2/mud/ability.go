package main

// ability.go — JOB_SPELL_SYSTEM_NORTHSTAR.md §1, Phase 1's remaining scope: "one data shape for
// everything a player can activate — a job ability (today's /ja) and a spell (today's cast)
// become the same underlying thing, differing only in field values." Phase 0/the universal
// lockout/the real Berserk-Boost-Sneak-Attack effects (all shipped 2026-09-12) already covered
// Phase 1's own lockout and pending-effect slices; this file covers the one real piece those
// didn't touch: MP cost and CAST TIME as one shared, data-driven model, plus the deferred-cast
// mechanism a real cast time actually requires (nothing in this codebase waited on a timer before
// this file — every spell resolved the instant it was typed, gated only by the universal lockout).
//
// Real, deliberate, bounded scope for THIS slice, named rather than silently assumed: this does
// NOT literally collapse cmdJA/cmdCast into one cmdAbility(id, target) entry point the way §1's
// own prose example shows — every current job ability (§4's roster) already has CastTime=0 (case
// 2 in §1: "today's JAs already work this way, no change needed to this case's own mechanics"),
// so the real, felt gap is entirely on the spell side. Building the full merged dispatcher too
// would mean touching cmdJA's own real, already-tested effect switch for zero behavioral gain
// this pass — named as real, remaining follow-up if byte-for-byte doc compliance is wanted later,
// not silently dropped. What IS real and shared here: the MP-cost/cast-time data model, and
// checkNotCasting, which both cmdCast and cmdJA call — casting a spell blocks a JA attempt and
// vice versa, one real shared action economy, not two independent ones that happen to coexist.
//
// Also NOT attempted here (both are named, open, unresolved questions in the NORTHSTAR doc
// itself, §"Real open questions" #4): interruption on taking damage while casting, and refunding
// MP on a failed/interrupted cast. A cast in progress today simply always completes once started
// (short of the caster disconnecting) — real, honest, deliberately not FFXI-accurate on this one
// point, matching the doc's own "simplified (no interrupts for now, revisit later)" framing as
// the one this slice actually ships.

import (
	"fmt"
	"time"
)

// AbilityKind (§3's own three real targeting buckets, named as a real enum rather than left as
// scattered per-case job-gate/target logic) is currently informational only in this slice — it
// documents which of §3's targeting rules a given ability/spell falls under, but the actual
// enforcement (auto-target-if-set for offensive, always-explicit for defensive) still lives in
// each real case's own existing logic (cmdCastBlackMagic's TargetMobID check, resolveSpellTarget
// for buffs/heals) rather than being newly generalized here — a real, separate, not-yet-built
// piece of §3's own scope, distinct from this file's MP-cost/cast-time focus.
type AbilityKind int

const (
	AbilityOffensive AbilityKind = iota // auto-targets p.combat.TargetMobID, no argument required
	AbilityDefensive                    // always requires an explicit target argument (heals/buffs)
	AbilitySelfOnly                     // never takes a target argument
)

// spellCastTimes gives every real, already-shipped spell for the three casting jobs among the
// six currently-enabled ones (WHM/RDM/BLM — job.WAR/MNK/THF's own real abilities are all §1 case
// 2, CastTime=0, unchanged) a real, non-zero cast time for the first time. Values are a real,
// deliberate v0 tuning choice — a short-tier-nuke/DoT/single-target-heal band at 2.0s, a
// stronger/tier-2 band around 2.5-2.75s, and the strongest tier-3 nukes plus longer-duration
// party buffs at 3.0-3.5s — NOT lifted from FFXI's own real cast-time table (this game's whole
// stat/level curve is already its own, separate thing per JOB_SPELL_SYSTEM_NORTHSTAR.md §5), and
// real, named, obviously tunable rather than treated as final. A spell with no entry here keeps
// CastTime=0 (general utility everyone has — invisible/sneak — or content belonging to a
// currently-disabled job — paladin/dark magic/ninjutsu/bard songs/teleport — untouched on
// purpose, see JOB_SPELL_SYSTEM_NORTHSTAR.md §6 Phase 0).
var spellCastTimes = map[string]time.Duration{
	"cure":  2 * time.Second,
	"cure2": time.Duration(2500) * time.Millisecond,

	"protect": 3 * time.Second,
	"shell":   3 * time.Second,
	"haste":   3 * time.Second,
	"regen":   3 * time.Second,
	"refresh": 3 * time.Second,

	"dia": 2 * time.Second,

	// blmSpells' own full roster (poison/bio/distract/frazzle share Poison's numbers per that
	// map's own doc comment; every element gets the same real tier-based band).
	"poison": 2 * time.Second, "bio": 2 * time.Second,
	"distract": 2 * time.Second, "frazzle": 2 * time.Second,
	"fire": 2 * time.Second, "fire2": time.Duration(2750) * time.Millisecond, "fire3": time.Duration(3500) * time.Millisecond,
	"blizzard": 2 * time.Second, "blizzard2": time.Duration(2750) * time.Millisecond, "blizzard3": time.Duration(3500) * time.Millisecond,
	"thunder": 2 * time.Second, "thunder2": time.Duration(2750) * time.Millisecond, "thunder3": time.Duration(3500) * time.Millisecond,
	"stone": 2 * time.Second, "stone2": time.Duration(2750) * time.Millisecond, "stone3": time.Duration(3500) * time.Millisecond,
	"water": 2 * time.Second, "water2": time.Duration(2750) * time.Millisecond, "water3": time.Duration(3500) * time.Millisecond,
	"aero": 2 * time.Second, "aero2": time.Duration(2750) * time.Millisecond, "aero3": time.Duration(3500) * time.Millisecond,
}

// spellMPCosts mirrors the exact real MP-cost constants already inline in cmdCast/blmSpells --
// duplicated here (not extracted as the single source of truth those call sites read from) so
// this slice touches zero existing case bodies, keeping the actual effect-application logic
// completely unchanged and low-risk. A real, named follow-up: once the full Ability-struct
// unification lands, these numbers should live in exactly one place, not two in sync by
// convention -- tracked in JOB_SPELL_SYSTEM_NORTHSTAR.md, not silently left as a maintenance trap.
var spellMPCosts = map[string]int{
	"cure": 50, "cure2": 80,
	"protect": 60, "shell": 60, "haste": 75, "regen": 40, "refresh": 50,
	"dia":    30,
	"poison": 30, "bio": 30, "distract": 30, "frazzle": 30,
	"fire": 30, "fire2": 65, "fire3": 120,
	"blizzard": 30, "blizzard2": 65, "blizzard3": 120,
	"thunder": 35, "thunder2": 75, "thunder3": 140,
	"stone": 25, "stone2": 55, "stone3": 100,
	"water": 28, "water2": 60, "water3": 110,
	"aero": 27, "aero2": 58, "aero3": 105,
}

// spellRequiresMobTarget lists every spell whose real effect (cmdCastBlackMagic's own check, and
// the "dia" case in cmdCast) requires a live p.combat.TargetMobID -- used only for the cheap
// preflight below, mirroring (not replacing) each case's own real check at completion time.
var spellRequiresMobTarget = map[string]bool{
	"dia":    true,
	"poison": true, "bio": true, "distract": true, "frazzle": true,
	"fire": true, "fire2": true, "fire3": true,
	"blizzard": true, "blizzard2": true, "blizzard3": true,
	"thunder": true, "thunder2": true, "thunder3": true,
	"stone": true, "stone2": true, "stone3": true,
	"water": true, "water2": true, "water3": true,
	"aero": true, "aero2": true, "aero3": true,
}

// spellRequiresAllyTarget lists every buff/heal whose real effect resolves a target via
// resolveSpellTarget (which already defaults to the caster itself on an empty/self name -- this
// preflight can only ever fail here on a real bad explicit name, e.g. a typo'd party member).
var spellRequiresAllyTarget = map[string]bool{
	"cure": true, "cure2": true, "protect": true, "shell": true,
	"haste": true, "regen": true, "refresh": true,
}

// spellDisplayNames gives a real, human-readable name for the "You begin casting ..." message --
// falls back to the raw spell id (title-cased is more effort than this cosmetic message needs)
// for anything not listed, so adding a new cast-time spell to spellCastTimes above without also
// updating this map degrades gracefully instead of breaking.
var spellDisplayNames = map[string]string{
	"cure": "Cure", "cure2": "Cure II", "protect": "Protect", "shell": "Shell",
	"haste": "Haste", "regen": "Regen", "refresh": "Refresh", "dia": "Dia",
	"poison": "Poison", "bio": "Bio", "distract": "Distract", "frazzle": "Frazzle",
	"fire": "Fire", "fire2": "Fire II", "fire3": "Fire III",
	"blizzard": "Blizzard", "blizzard2": "Blizzard II", "blizzard3": "Blizzard III",
	"thunder": "Thunder", "thunder2": "Thunder II", "thunder3": "Thunder III",
	"stone": "Stone", "stone2": "Stone II", "stone3": "Stone III",
	"water": "Water", "water2": "Water II", "water3": "Water III",
	"aero": "Aero", "aero2": "Aero II", "aero3": "Aero III",
}

func spellDisplayName(spell string) string {
	if name, ok := spellDisplayNames[spell]; ok {
		return name
	}
	return spell
}

// checkNotCasting is the real shared gate cmdCast and cmdJA both call (§1's own "one shared
// action economy" requirement, the part of the unification that actually matters while every JA
// still has CastTime=0): refuses a new spell/ability attempt while a previous cast is still
// resolving. Does NOT touch/extend lastActionAt (the universal lockout) -- that is a separate,
// already-independent timer per its own doc comment.
func (p *player) checkNotCasting() bool {
	if p.castingSpell == "" || !time.Now().Before(p.castingCompletesAt) {
		return true
	}
	remaining := time.Until(p.castingCompletesAt)
	p.sendf("You are already casting %s. (%s remaining)", spellDisplayName(p.castingSpell), remaining.Round(100*time.Millisecond))
	p.prompt()
	return false
}

// castPreflight runs the cheap, real checks that are worth failing FAST on -- before a player
// stands still for a multi-second cast bar only to be told it was never going to work -- without
// duplicating each spell's own real effect/validation logic (that still runs again, unchanged, at
// completion time in castNow; this is purely an early, honest "would this even work" gate, not a
// second source of truth for MP deduction or target resolution). A real, named, minor gap: job
// gate (does the caster's job/subjob even grant this spell) is NOT preflighted here -- casting a
// spell your job can't use still waits out the full cast bar before castNow's own existing job
// check refuses it. Low-frequency in practice (a player mostly only ever types spells their own
// job actually has) and not worth duplicating every one of castNow's own per-spell job-gate
// checks a second time for this slice.
func castPreflight(p *player, spell, targetName string) (ok bool, failMsg string) {
	if cost, known := spellMPCosts[spell]; known && p.mp < cost {
		return false, fmt.Sprintf("Not enough MP. (need %d, have %d)", cost, p.mp)
	}
	if spellRequiresMobTarget[spell] {
		if p.combat.TargetMobID == "" {
			return false, "No target. Use 'target <mob>' first."
		}
	}
	if spellRequiresAllyTarget[spell] {
		if _, errMsg := resolveSpellTarget(p, targetName); errMsg != "" {
			return false, errMsg
		}
	}
	return true, ""
}

// beginCast is cmdCast's own real entry point for a spell with CastTime > 0 (§1 case 3, "real
// cast time"): validates via castPreflight (fails immediately, consumes nothing -- the universal
// lockout's own p.lastActionAt was already stamped by cmdCast's own checkUniversalLockout call
// before this runs, matching every other failed-cast-attempt's own existing behavior, not a new
// exemption), then defers the EXACT existing castNow effect logic (unchanged, still doing its own
// real job-gate/MP-deduction/target-refetch at completion time) via time.AfterFunc.
//
// The completion callback runs on its own goroutine (time.AfterFunc's own real contract) and so
// does NOT hold gw.mu the way cmdCast's own caller (handle()) does -- it must acquire it itself,
// matching runHeadlessCommand's own established pattern for a background goroutine touching
// player/world state outside handle()'s own call chain. Re-looks-up the player by slot (rather
// than closing over the *player pointer directly) and checks it's still THIS cast still pending
// before doing anything -- a disconnected player is simply never in gw.players anymore, and this
// silently no-ops rather than writing to a dead connection or a struct nothing else references.
func beginCast(p *player, spell, targetName string, castTime time.Duration) {
	if ok, failMsg := castPreflight(p, spell, targetName); !ok {
		p.send(failMsg)
		p.prompt()
		return
	}
	p.castingSpell = spell
	p.castingCompletesAt = time.Now().Add(castTime)
	p.sendf("You begin casting %s... (%s)", spellDisplayName(spell), castTime.Round(100*time.Millisecond))
	p.prompt()

	slot := p.slot
	time.AfterFunc(castTime, func() {
		gw.mu.Lock()
		defer gw.mu.Unlock()
		pp, ok := gw.players[slot]
		if !ok || pp.castingSpell != spell {
			return // disconnected, or this cast was already cleared/superseded
		}
		pp.castingSpell = ""
		pp.castingCompletesAt = time.Time{}
		castNow(pp, spell, targetName)
	})
}
