package system

import "testing"

func nearlyEqual(a, b, tol float64) bool {
	if a > b {
		return a-b <= tol
	}
	return b-a <= tol
}

func TestStepBoidsAlignment(t *testing.T) {
	boids := []BoidState{
		{Pos: Vec3{X: 0, Y: 0, Z: 0}, Vel: Vec3{X: 1, Y: 0, Z: 0}},
		{Pos: Vec3{X: 1, Y: 0, Z: 0}, Vel: Vec3{X: 0, Y: 0, Z: 1}},
	}
	cfg := BoidConfig{
		NeighborRadius:  5,
		AlignmentWeight: 1,
		MaxSpeed:        1,
	}

	updated := StepBoids(boids, cfg)
	vel := updated[0].Vel
	if vel.X <= 0 || vel.Z <= 0 {
		t.Fatalf("expected alignment to steer toward positive x/z, got %+v", vel)
	}
	if !nearlyEqual(vec3Len(vel), 1, 1e-6) {
		t.Fatalf("expected velocity to be normalized to max speed")
	}
}

func TestStepBoidsSeparation(t *testing.T) {
	boids := []BoidState{
		{Pos: Vec3{X: 0, Y: 0, Z: 0}, Vel: Vec3{X: 1, Y: 0, Z: 0}},
		{Pos: Vec3{X: 0.2, Y: 0, Z: 0}, Vel: Vec3{X: 0, Y: 0, Z: 1}},
	}
	cfg := BoidConfig{
		NeighborRadius:   5,
		SeparationRadius: 1,
		SeparationWeight: 1,
		MaxSpeed:         1,
	}

	updated := StepBoids(boids, cfg)
	vel := updated[0].Vel
	if vel.X >= 0 {
		t.Fatalf("expected separation to push away on negative x, got %+v", vel)
	}
}

func TestStepBoidsCohesion(t *testing.T) {
	boids := []BoidState{
		{Pos: Vec3{X: 0, Y: 0, Z: 0}, Vel: Vec3{X: 0, Y: 0, Z: 0}},
		{Pos: Vec3{X: 4, Y: 0, Z: 0}, Vel: Vec3{X: 0, Y: 0, Z: 0}},
	}
	cfg := BoidConfig{
		NeighborRadius: 5,
		CohesionWeight: 1,
		MaxSpeed:       1,
	}

	updated := StepBoids(boids, cfg)
	vel := updated[0].Vel
	if vel.X <= 0 {
		t.Fatalf("expected cohesion to move toward neighbor on positive x, got %+v", vel)
	}
}
