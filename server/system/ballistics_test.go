package system

import "testing"

func TestDirectionFromYawPitch(t *testing.T) {
	vec := DirectionFromYawPitch(0, 0)
	if vec.X < -0.0001 || vec.X > 0.0001 {
		t.Fatalf("expected X near 0, got %f", vec.X)
	}
	if vec.Y < -0.0001 || vec.Y > 0.0001 {
		t.Fatalf("expected Y near 0, got %f", vec.Y)
	}
	if vec.Z < 0.999 || vec.Z > 1.001 {
		t.Fatalf("expected Z near 1, got %f", vec.Z)
	}

	vec = DirectionFromYawPitch(-90, 0)
	if vec.X < 0.999 || vec.X > 1.001 {
		t.Fatalf("expected X near 1, got %f", vec.X)
	}
	if vec.Z < -0.0001 || vec.Z > 0.0001 {
		t.Fatalf("expected Z near 0, got %f", vec.Z)
	}
}
