package main

import (
	"testing"

	combatTp "dragonsnshit/server/combat"
	"dragonsnshit/server/player"
	"dragonsnshit/server/system"
)

// TestGameWorldRayTrace_HitsTargetOnAxis checks the real, ported-from-SHANKPIT sphere
// intersection: a ray fired straight at a target standing directly ahead should hit.
func TestGameWorldRayTrace_HitsTargetOnAxis(t *testing.T) {
	gw := &gameWorld{
		shooterID: 1,
		clients: map[string]clientInfo{
			"shooter": {id: 1, pos: system.Vec3{X: 0, Y: 0, Z: 0}},
			"target":  {id: 2, pos: system.Vec3{X: 0, Y: 0, Z: 10}},
		},
	}
	start := system.Vec3{X: 0, Y: 0.9, Z: 0} // chest height, matching gameWorld's own +0.9 target offset
	end := system.Vec3{X: 0, Y: 0.9, Z: 100}
	res, hit := gw.RayTrace(start, end)
	if !hit {
		t.Fatal("expected a hit on a target directly ahead, got no hit")
	}
	entityHit, ok := res.(gameEntityHit)
	if !ok {
		t.Fatalf("expected a gameEntityHit result, got %T", res)
	}
	if entityHit.clientID != 2 {
		t.Fatalf("expected clientID=2, got %d", entityHit.clientID)
	}
}

func TestGameWorldRayTrace_MissesTargetOffAxis(t *testing.T) {
	gw := &gameWorld{
		shooterID: 1,
		clients: map[string]clientInfo{
			"shooter": {id: 1, pos: system.Vec3{X: 0, Y: 0, Z: 0}},
			"target":  {id: 2, pos: system.Vec3{X: 20, Y: 0, Z: 10}}, // well off the ray's own path
		},
	}
	start := system.Vec3{X: 0, Y: 0.9, Z: 0}
	end := system.Vec3{X: 0, Y: 0.9, Z: 100}
	_, hit := gw.RayTrace(start, end)
	if hit {
		t.Fatal("expected no hit on a target far off the ray's own axis")
	}
}

func TestGameWorldRayTrace_NeverHitsSelf(t *testing.T) {
	gw := &gameWorld{
		shooterID: 1,
		clients: map[string]clientInfo{
			"shooter": {id: 1, pos: system.Vec3{X: 0, Y: 0, Z: 5}}, // sitting right on the ray's own path
		},
	}
	start := system.Vec3{X: 0, Y: 0.9, Z: 0}
	end := system.Vec3{X: 0, Y: 0.9, Z: 100}
	_, hit := gw.RayTrace(start, end)
	if hit {
		t.Fatal("expected the shooter's own client to never be a valid hit target")
	}
}

func TestGameWorldRayTrace_IgnoresTargetsBehindShooter(t *testing.T) {
	gw := &gameWorld{
		shooterID: 1,
		clients: map[string]clientInfo{
			"shooter": {id: 1, pos: system.Vec3{X: 0, Y: 0, Z: 0}},
			"behind":  {id: 2, pos: system.Vec3{X: 0, Y: 0, Z: -10}}, // behind the ray's own origin
		},
	}
	start := system.Vec3{X: 0, Y: 0.9, Z: 0}
	end := system.Vec3{X: 0, Y: 0.9, Z: 100}
	_, hit := gw.RayTrace(start, end)
	if hit {
		t.Fatal("expected a target behind the shooter to never be hit")
	}
}

func TestGameWorldRayTrace_IgnoresTargetsBeyondRange(t *testing.T) {
	gw := &gameWorld{
		shooterID: 1,
		clients: map[string]clientInfo{
			"shooter": {id: 1, pos: system.Vec3{X: 0, Y: 0, Z: 0}},
			"faraway": {id: 2, pos: system.Vec3{X: 0, Y: 0, Z: 500}}, // past the ray's own end point
		},
	}
	start := system.Vec3{X: 0, Y: 0.9, Z: 0}
	end := system.Vec3{X: 0, Y: 0.9, Z: 100}
	_, hit := gw.RayTrace(start, end)
	if hit {
		t.Fatal("expected a target beyond the ray's own max range to never be hit")
	}
}

func TestGameWorldRayTrace_PicksClosestOfMultipleTargets(t *testing.T) {
	gw := &gameWorld{
		shooterID: 1,
		clients: map[string]clientInfo{
			"shooter": {id: 1, pos: system.Vec3{X: 0, Y: 0, Z: 0}},
			"far":     {id: 2, pos: system.Vec3{X: 0, Y: 0, Z: 30}},
			"near":    {id: 3, pos: system.Vec3{X: 0, Y: 0, Z: 10}},
		},
	}
	start := system.Vec3{X: 0, Y: 0.9, Z: 0}
	end := system.Vec3{X: 0, Y: 0.9, Z: 100}
	res, hit := gw.RayTrace(start, end)
	if !hit {
		t.Fatal("expected a hit")
	}
	entityHit := res.(gameEntityHit)
	if entityHit.clientID != 3 {
		t.Fatalf("expected the nearer target (id=3) to be picked over the farther one (id=2), got id=%d",
			entityHit.clientID)
	}
}

func TestGameWorldRayTrace_ZeroLengthRayNoHit(t *testing.T) {
	gw := &gameWorld{
		shooterID: 1,
		clients: map[string]clientInfo{
			"target": {id: 2, pos: system.Vec3{X: 0, Y: 0, Z: 0}},
		},
	}
	_, hit := gw.RayTrace(system.Vec3{}, system.Vec3{})
	if hit {
		t.Fatal("expected a zero-length ray to never report a hit")
	}
}

// TestGameEntityHit_EntityAppliesRealDamage checks the real fix (founder: "for damage we want
// to make it match up") -- Entity().Hurt() now actually reduces the hit client's own real
// hpState, mirroring PacketWSCast's own already-real damage application, mutating the same
// clients map the caller (main loop) owns since Go maps are reference types.
func TestGameEntityHit_EntityAppliesRealDamage(t *testing.T) {
	clients := map[string]clientInfo{
		"target": {id: 2, hpState: combatTp.NewHPState(100)},
	}
	hit := gameEntityHit{clientID: 2, slot: "target", clients: clients}
	hit.Entity().Hurt(30, player.DamageSource{Cause: player.CauseProjectile})
	if got := clients["target"].hpState.Current; got != 70 {
		t.Fatalf("expected 100-30=70 HP after Hurt(30), got %d", got)
	}
}

func TestGameEntityHit_EntityHurtOnMissingClientIsSafeNoop(t *testing.T) {
	clients := map[string]clientInfo{}
	hit := gameEntityHit{clientID: 99, slot: "ghost", clients: clients}
	hit.Entity().Hurt(30, player.DamageSource{Cause: player.CauseProjectile}) // must not panic
}

func TestGameEntityHit_EntityHurtCreatesFallbackHPStateIfMissing(t *testing.T) {
	clients := map[string]clientInfo{
		"target": {id: 2}, // no hpState -- real defensive path, same as PacketWSCast's own
	}
	hit := gameEntityHit{clientID: 2, slot: "target", clients: clients}
	hit.Entity().Hurt(5, player.DamageSource{Cause: player.CauseProjectile})
	if clients["target"].hpState == nil {
		t.Fatal("expected a fallback hpState to be created, got nil")
	}
}
