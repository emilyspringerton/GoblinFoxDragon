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

func TestVec3_Sub(t *testing.T) {
	got := Vec3{X: 5, Y: 3, Z: 1}.Sub(Vec3{X: 2, Y: 1, Z: 4})
	want := Vec3{X: 3, Y: 2, Z: -3}
	if got != want {
		t.Fatalf("Sub: got %+v, want %+v", got, want)
	}
}

func TestVec3_Dot(t *testing.T) {
	got := Vec3{X: 1, Y: 2, Z: 3}.Dot(Vec3{X: 4, Y: 5, Z: 6})
	want := 32.0 // 1*4 + 2*5 + 3*6
	if got != want {
		t.Fatalf("Dot: got %f, want %f", got, want)
	}
	// Perpendicular vectors dot to zero -- the real property RayTrace's own closest-point math
	// relies on.
	if perp := (Vec3{X: 1, Y: 0, Z: 0}).Dot(Vec3{X: 0, Y: 1, Z: 0}); perp != 0 {
		t.Fatalf("expected perpendicular vectors to dot to 0, got %f", perp)
	}
}

func TestVec3_Len(t *testing.T) {
	got := Vec3{X: 3, Y: 4, Z: 0}.Len()
	if got != 5 {
		t.Fatalf("Len: got %f, want 5 (real 3-4-5 triangle)", got)
	}
	if (Vec3{}).Len() != 0 {
		t.Fatal("expected zero vector to have length 0")
	}
}
