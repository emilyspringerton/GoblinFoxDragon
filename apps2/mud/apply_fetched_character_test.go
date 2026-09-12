package main

import (
	"testing"

	"dragonsnshit/server/homepoint"
	"dragonsnshit/server/idunaclient"
)

// TestRestoreHomePointFromCharacter_RestoresARealHome guards the real, founder-reported live gap
// (2026-09-12): "home point does not persist after logouts." getOrCreateHeadlessPlayer's own
// connect path already restored a returning character's real, IDUNA-persisted home point
// (2026-08-04 fix) -- applyFetchedCharacter, the function every REAL SSH reconnect actually goes
// through (handleConn's preset != nil branch), never did: a `sethome` set via SSH was correctly
// PERSISTED (idunaclient.UpdateHome) but never LOADED BACK on the next connection, since
// applyFetchedCharacter never touched p.homePoint at all. Both real call sites now share this
// one function instead of two copies that already once drifted apart.
func TestRestoreHomePointFromCharacter_RestoresARealHome(t *testing.T) {
	p := newTestPlayer()
	p.homePoint = homepoint.NewState(0)
	ch := &idunaclient.Character{HomeSceneID: 2, HomePosX: 10, HomePosY: 5, HomePosZ: -3}

	restoreHomePointFromCharacter(p, ch)

	if p.homePoint.Home == nil {
		t.Fatal("expected a home point to be set, got nil")
	}
	if p.homePoint.Home.SceneID != 2 || p.homePoint.Home.Pos.X != 10 || p.homePoint.Home.Pos.Y != 5 || p.homePoint.Home.Pos.Z != -3 {
		t.Errorf("home point = %+v, want SceneID=2 Pos={10,5,-3}", p.homePoint.Home)
	}
}

// TestRestoreHomePointFromCharacter_NoHomeSetLeavesItUnset guards the real, named, accepted edge
// case: a character record whose home fields are all still the literal zero-value default
// (SceneID 0, position 0,0,0 -- IDUNA's own default row, no is-set flag exists in the schema)
// must NOT be treated as "a home is set at the origin of Meadow" -- it must leave p.homePoint
// alone (still unset), matching getOrCreateHeadlessPlayer's own identical guard.
func TestRestoreHomePointFromCharacter_NoHomeSetLeavesItUnset(t *testing.T) {
	p := newTestPlayer()
	p.homePoint = homepoint.NewState(0)
	ch := &idunaclient.Character{} // every Home* field at its real zero value

	restoreHomePointFromCharacter(p, ch)

	if p.homePoint.Home != nil {
		t.Errorf("expected no home point to be set for an all-zero character record, got %+v", p.homePoint.Home)
	}
}
