package player

import (
	"testing"

	"dragonsnshit/server/system"
)

type testWorld struct {
	lastStart system.Vec3
	lastEnd   system.Vec3
	result    RaycastResult
	found     bool
}

func (w *testWorld) RayTrace(start, end system.Vec3) (RaycastResult, bool) {
	w.lastStart = start
	w.lastEnd = end
	return w.result, w.found
}

type testEntity struct {
	hurtAmount float64
	hurted     bool
}

func (e *testEntity) Hurt(amount float64, _ DamageSource) {
	e.hurted = true
	e.hurtAmount = amount
}

type testEntityResult struct {
	pos    system.Vec3
	entity LivingEntity
}

func (r testEntityResult) Position() system.Vec3 { return r.pos }
func (r testEntityResult) Entity() LivingEntity  { return r.entity }

type testBlockResult struct {
	pos system.Vec3
}

func (r testBlockResult) Position() system.Vec3 { return r.pos }

type testPlayer struct {
	pos          system.Vec3
	eyeHeight    float64
	world        *testWorld
	lastSound    string
	lastSoundPos system.Vec3
}

func (p *testPlayer) Position() system.Vec3 { return p.pos }
func (p *testPlayer) EyeHeight() float64    { return p.eyeHeight }
func (p *testPlayer) World() RaycastWorld   { return p.world }
func (p *testPlayer) SendSound(name string, pos system.Vec3) {
	p.lastSound = name
	p.lastSoundPos = pos
}

func TestHandleShankFireEntityHit(t *testing.T) {
	entity := &testEntity{}
	world := &testWorld{
		result: testEntityResult{pos: system.Vec3{X: 1, Y: 2, Z: 3}, entity: entity},
		found:  true,
	}
	player := &testPlayer{pos: system.Vec3{}, eyeHeight: 1.6, world: world}

	hit, pos, hitEntity := HandleShankFire(player, 0, 0, 1)

	if !entity.hurted {
		t.Fatalf("expected entity to be hurt")
	}
	if entity.hurtAmount != 50.0 {
		t.Fatalf("expected 50 damage, got %f", entity.hurtAmount)
	}
	if player.lastSound != "random.orb" {
		t.Fatalf("expected hit sound, got %s", player.lastSound)
	}
	if !hit || !hitEntity {
		t.Fatalf("expected entity hit result, got hit=%v entity=%v", hit, hitEntity)
	}
	if pos != (system.Vec3{X: 1, Y: 2, Z: 3}) {
		t.Fatalf("expected hit position to match entity hit")
	}
	if world.lastEnd.Z < system.MagnumRange-0.01 || world.lastEnd.Z > system.MagnumRange+0.01 {
		t.Fatalf("expected magnum range end.z near %f, got %f", system.MagnumRange, world.lastEnd.Z)
	}
}

func TestHandleShankFireBlockHit(t *testing.T) {
	world := &testWorld{
		result: testBlockResult{pos: system.Vec3{X: 4, Y: 5, Z: 6}},
		found:  true,
	}
	player := &testPlayer{pos: system.Vec3{}, eyeHeight: 1.6, world: world}

	hit, _, hitEntity := HandleShankFire(player, 0, 0, 0)

	if player.lastSound != "step.wood" {
		t.Fatalf("expected wood sound, got %s", player.lastSound)
	}
	if !hit || hitEntity {
		t.Fatalf("expected block hit result, got hit=%v entity=%v", hit, hitEntity)
	}
}
