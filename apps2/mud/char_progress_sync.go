package main

// char_progress_sync.go fixes a real, confirmed, live production data-loss bug (2026-09-12,
// founder real-time: "ok we lost data somehow - did you roll the server back?" / "i was lvl 11
// warrior before losing connection" -- resolved to level 8 on reconnect / "this is unacceptable
// we can never lose player progress like that").
//
// Root cause, confirmed by direct code reading: a real telnet/SSH player's level/XP/Flow was
// ONLY ever persisted to IDUNA in handleConn's own disconnect-time `defer` block (grep
// "S98-02: save level/xp on disconnect"). That defer only runs when the connection's own
// goroutine unwinds normally -- this process has NO signal.Notify(SIGTERM)/graceful-shutdown
// handling anywhere (checked directly: the only real signal handler in this whole binary is
// hot_reload.go's own SIGHUP watcher), so `systemctl restart gfd-mud.service` sends the default
// SIGTERM, Go's default unhandled-SIGTERM behavior kills the process immediately, and every
// in-flight connection's disconnect defer is skipped entirely -- ANY level/XP/Flow progress
// earned since the player's LAST successful save is silently lost on every restart, not just an
// unlucky one. This exactly explains the founder's report: WAR's last real saved checkpoint was
// level 8 (see character_job_levels), and whatever levels were earned after that point, before
// the next restart, never made it into a save.
//
// The real fix mirrors S252's own already-proven headless-session sync pattern (see
// headlessSyncedLevel/XP/Flow's own doc comment on the player struct) -- a periodic, delta-only
// sync -- but applies it to EVERY player, not only headless ones: called once per tickAll() (1Hz)
// for every connected player, so at most ~1 second of progress can ever be lost to an abrupt
// process death, not potentially hours of unsaved play. Deliberately does NOT touch inventory
// sync (a separate, actively in-progress change elsewhere in this file) -- scoped narrowly to the
// exact regression just found: level/XP/Flow.

// needsProgressBaseline reports whether p has never been through syncCharLevelAndFlow before.
// xp.MinLevel is 1 (never 0 for a real character, checked directly in server/xp), so
// headlessSyncedLevel == 0 is a safe, unambiguous "never yet baselined" sentinel. Real telnet/SSH
// connect paths (unlike getOrCreateHeadlessPlayer) never initialize headlessSyncedLevel/XP/Flow
// at connect time -- without this baseline step, such a player's very first sync call would see a
// spurious "delta" (headlessSyncedFlow's real Go zero value, 0, vs. the player's real current
// Flow) and double-credit their real balance via CreditGold.
func needsProgressBaseline(p *player) bool {
	return p.headlessSyncedLevel == 0
}

// baselineCharProgress captures p's current level/XP/Flow as the sync baseline with no write --
// exactly mirroring what getOrCreateHeadlessPlayer already does explicitly for headless sessions.
func baselineCharProgress(p *player) {
	p.headlessSyncedLevel = p.charXP.Level
	p.headlessSyncedXP = p.charXP.CurrentXP
	p.headlessSyncedFlow = p.flow
}

// needsLevelSync reports whether p's level or XP has changed since the last real sync.
func needsLevelSync(p *player) bool {
	return p.charXP.Level != p.headlessSyncedLevel || p.charXP.CurrentXP != p.headlessSyncedXP
}

// flowSyncDelta returns p's real Flow change since the last sync (0 = nothing to sync).
func flowSyncDelta(p *player) int {
	return p.flow - p.headlessSyncedFlow
}

// syncCharLevelAndFlow persists p's level/XP/Flow to IDUNA if either has changed since the last
// sync -- safe to call every tick for every player: the delta checks above make a no-op the
// overwhelmingly common case, matching the exact real, already-proven headless convention this
// generalizes to every player regardless of transport.
func syncCharLevelAndFlow(p *player) {
	charID := gw.charIDBySlot[p.slot]
	if charID == "" {
		return
	}
	if needsProgressBaseline(p) {
		baselineCharProgress(p)
		return
	}
	if needsLevelSync(p) {
		_ = gw.iduna.UpdateCharacterLevel(charID, p.charXP.Level, p.charXP.CurrentXP)
		// GFD-124433: also persist to the currently-active JOB's own row -- characters.level
		// stays a "whichever job is active right now" mirror for anything else reading a
		// character's plain level; character_job_levels is the real, per-job source of truth.
		_ = gw.iduna.UpdateJobLevel(charID, p.jobID, p.charXP.Level, p.charXP.CurrentXP)
		p.headlessSyncedLevel = p.charXP.Level
		p.headlessSyncedXP = p.charXP.CurrentXP
	}
	if delta := flowSyncDelta(p); delta != 0 {
		if delta > 0 {
			_ = gw.iduna.CreditGold(charID, delta)
		} else {
			_ = gw.iduna.DeductGold(charID, -delta)
		}
		p.headlessSyncedFlow = p.flow
	}
}

// miningSkillDelta/fishingSkillDelta return p's real gather-skill change since the last sync
// (0 = nothing to sync). Unlike level (xp.MinLevel is never 0), a genuinely untrained skill IS
// 0.0, so there is no "never baselined" sentinel here -- every real connect path instead
// explicitly sets syncedMiningSkill/syncedFishingSkill to match whatever was just loaded from
// IDUNA (see getOrCreateHeadlessPlayer/handleConn), so these deltas are naturally 0 until real,
// new gathering happens this session.
func miningSkillDelta(p *player) float64  { return p.miningSkill - p.syncedMiningSkill }
func fishingSkillDelta(p *player) float64 { return p.fishingSkill - p.syncedFishingSkill }

// syncGatherSkills persists p's mining/fishing skill gains to IDUNA (founder real-time,
// 2026-09-12: "can we make sure fishing skill persists too?") -- the same real class of bug as
// level/XP/Flow (see this file's own top-of-file doc comment), just for a field that had NO
// persistence path at all, not even a disconnect-time save. IncrementSkill is delta-based
// (IDUNA's own real upsert adds delta to whatever's already stored), matching exactly what these
// deltas represent.
func syncGatherSkills(p *player) {
	charID := gw.charIDBySlot[p.slot]
	if charID == "" {
		return
	}
	if delta := miningSkillDelta(p); delta != 0 {
		_ = gw.iduna.IncrementSkill(charID, "mining", delta)
		p.syncedMiningSkill = p.miningSkill
	}
	if delta := fishingSkillDelta(p); delta != 0 {
		_ = gw.iduna.IncrementSkill(charID, "fishing", delta)
		p.syncedFishingSkill = p.fishingSkill
	}
}
