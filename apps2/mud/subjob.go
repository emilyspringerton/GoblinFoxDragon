package main

import (
	"dragonsnshit/server/job"
)

// subJobMinLevel is the character level at which a sub job becomes available (FFXI: level 18).
const subJobMinLevel = 18

// syncCharJobLevels keeps p.charJob's levels in step with the character's level. They were set once
// at login / `setsubjob` and went stale on every level-up, freezing combined stats and sub-job
// spell access at the old level.
func syncCharJobLevels(p *player) {
	if p.charJob == nil || p.charXP == nil {
		return
	}
	p.charJob.Main = job.JobID(p.jobID)
	p.charJob.MainLvl = p.charXP.Level
	if p.charJob.Sub != "" {
		p.charJob.SubLvl = p.charXP.Level // EffectiveSubLevel() halves this
	}
}

// subJobMP is the MP a sub job adds: its own MP pool at the effective (half) level.
func subJobMP(p *player) int {
	if p.charJob == nil || p.charJob.Sub == "" {
		return 0
	}
	mp, err := job.MPAtLevel(p.charJob.Sub, p.charJob.EffectiveSubLevel())
	if err != nil || mp < 0 {
		return 0
	}
	return mp
}

// abilitiesFor is every job ability the player can have recasts for: the main job's, plus the sub
// job's at the sub job's effective level (a sub ability with MinLevel N is available once the
// character is level 2N, because the sub job runs at half level).
func abilitiesFor(p *player) []job.Ability {
	out := abilitiesForJob(p.jobID)
	if p.charJob == nil || p.charJob.Sub == "" {
		return out
	}
	have := map[string]bool{}
	for _, a := range out {
		have[a.ID] = true
	}
	for _, a := range abilitiesForJob(string(p.charJob.Sub)) {
		if have[a.ID] {
			continue
		}
		a.MinLevel *= 2
		out = append(out, a)
	}
	return out
}
