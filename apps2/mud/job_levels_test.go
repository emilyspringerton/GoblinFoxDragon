package main

import (
	"testing"

	"dragonsnshit/server/idunaclient"
	"dragonsnshit/server/job"
	"dragonsnshit/server/xp"
)

// TestSwitchActiveJob_FirstTimeJobStartsAtLevelOne is the literal, real bug GFD-124433 reported:
// "when you are a lvl 10 warrior in gfd and you switch to RDM for the first time you go back to
// lvl 1."
func TestSwitchActiveJob_FirstTimeJobStartsAtLevelOne(t *testing.T) {
	jobXP := map[string]*xp.CharXP{
		job.WAR: {Level: 10, CurrentXP: 500},
	}
	rdm := switchActiveJob(jobXP, job.RDM)
	if rdm.Level != 1 || rdm.CurrentXP != 0 {
		t.Errorf("first-time RDM: got level=%d xp=%d, want level=1 xp=0", rdm.Level, rdm.CurrentXP)
	}
	if jobXP[job.WAR].Level != 10 {
		t.Errorf("switching to RDM must not touch WAR's own level, got %d", jobXP[job.WAR].Level)
	}
}

// TestSwitchActiveJob_ReturningToAPreviouslyLeveledJobKeepsItsLevel is the other half of
// GFD-124433: "...and switch back to your level 10 war."
func TestSwitchActiveJob_ReturningToAPreviouslyLeveledJobKeepsItsLevel(t *testing.T) {
	jobXP := map[string]*xp.CharXP{
		job.WAR: {Level: 10, CurrentXP: 500},
	}
	rdm := switchActiveJob(jobXP, job.RDM)
	rdm.Level = 5 // simulate leveling RDM up to 5, per the founder's own example
	rdm.CurrentXP = 42

	backToWar := switchActiveJob(jobXP, job.WAR)
	if backToWar.Level != 10 || backToWar.CurrentXP != 500 {
		t.Errorf("returning to WAR: got level=%d xp=%d, want level=10 xp=500", backToWar.Level, backToWar.CurrentXP)
	}

	// And switching back to RDM again must resume from level 5, not reset.
	rdmAgain := switchActiveJob(jobXP, job.RDM)
	if rdmAgain.Level != 5 || rdmAgain.CurrentXP != 42 {
		t.Errorf("returning to RDM: got level=%d xp=%d, want level=5 xp=42", rdmAgain.Level, rdmAgain.CurrentXP)
	}
	if rdmAgain != rdm {
		t.Error("expected the exact same *xp.CharXP pointer back, not a new one")
	}
}

func TestSwitchActiveJob_MutatesTheSameMapPassedIn(t *testing.T) {
	jobXP := map[string]*xp.CharXP{}
	switchActiveJob(jobXP, job.BLM)
	if _, ok := jobXP[job.BLM]; !ok {
		t.Error("expected switchActiveJob to add the new job's entry to the map in place")
	}
}

func TestLoadJobXP_SeedsMainJobFromLegacyColumnsWhenNoPersistedRowExists(t *testing.T) {
	// A character that predates this feature: characters.level=10, no character_job_levels rows
	// at all yet. Real, one-time backward-compat migration -- their real progress isn't lost.
	jobXP := loadJobXP(map[string]idunaclient.JobLevel{}, job.WAR, 10, 340)
	if jobXP[job.WAR] == nil || jobXP[job.WAR].Level != 10 || jobXP[job.WAR].CurrentXP != 340 {
		t.Errorf("expected WAR seeded from legacy columns, got %+v", jobXP[job.WAR])
	}
	if len(jobXP) != 1 {
		t.Errorf("expected exactly one seeded job, got %+v", jobXP)
	}
}

func TestLoadJobXP_PersistedRowsWinOverLegacyColumns(t *testing.T) {
	// Once a real per-job row exists for the main job, it's the ground truth -- legacy columns
	// (which apps2/mud keeps mirroring the active job into) must not override it.
	persisted := map[string]idunaclient.JobLevel{
		job.WAR: {Job: job.WAR, Level: 15, CurrentXP: 900},
	}
	jobXP := loadJobXP(persisted, job.WAR, 10, 340)
	if jobXP[job.WAR].Level != 15 {
		t.Errorf("expected the real persisted WAR level (15) to win over the legacy column (10), got %d", jobXP[job.WAR].Level)
	}
}

func TestLoadJobXP_LoadsEveryPersistedJobIndependently(t *testing.T) {
	persisted := map[string]idunaclient.JobLevel{
		job.WAR: {Job: job.WAR, Level: 10, CurrentXP: 500},
		job.RDM: {Job: job.RDM, Level: 5, CurrentXP: 42},
	}
	jobXP := loadJobXP(persisted, job.WAR, 10, 500)
	if len(jobXP) != 2 {
		t.Fatalf("expected both persisted jobs loaded, got %+v", jobXP)
	}
	if jobXP[job.WAR].Level != 10 || jobXP[job.RDM].Level != 5 {
		t.Errorf("unexpected levels: WAR=%d RDM=%d", jobXP[job.WAR].Level, jobXP[job.RDM].Level)
	}
}

func TestLoadJobXP_DoesNotEagerlyCreateUnplayedJobs(t *testing.T) {
	// A never-played job should be absent, not pre-created at level 1 -- switchActiveJob creates
	// it lazily the moment it's actually switched to, matching "starts at 1 the first time you
	// switch to it," not "every one of 23 jobs already has a phantom level-1 row."
	jobXP := loadJobXP(map[string]idunaclient.JobLevel{}, job.WAR, 10, 500)
	if _, ok := jobXP[job.BLM]; ok {
		t.Error("expected an unplayed job to be absent from the loaded map, not pre-created")
	}
}
