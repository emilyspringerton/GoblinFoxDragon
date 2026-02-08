package system

import "testing"

func TestGripForSlip(t *testing.T) {
	curve := TireGripCurve{
		PeakSlip:  0.12,
		PeakGrip:  1.4,
		SlideGrip: 0.9,
	}

	if got := curve.GripForSlip(0); got != 0 {
		t.Fatalf("expected zero grip at zero slip, got %.3f", got)
	}

	if got := curve.GripForSlip(0.12); got != curve.PeakGrip {
		t.Fatalf("expected peak grip at peak slip, got %.3f", got)
	}

	expectedSlide := curve.SlideGrip
	if got := curve.GripForSlip(0.4); got != expectedSlide {
		t.Fatalf("expected slide grip at large slip, got %.3f", got)
	}
}

func TestDownforce(t *testing.T) {
	aero := AeroModel{
		BaseDownforce:   1200,
		DownforcePerMS2: 3.5,
		MaxDownforce:    6000,
	}

	if got := aero.Downforce(0); got != aero.BaseDownforce {
		t.Fatalf("expected base downforce, got %.2f", got)
	}

	if got := aero.Downforce(50); got <= aero.BaseDownforce {
		t.Fatalf("expected increased downforce, got %.2f", got)
	}

	if got := aero.Downforce(100); got != aero.MaxDownforce {
		t.Fatalf("expected capped downforce, got %.2f", got)
	}
}

func TestLockupRisk(t *testing.T) {
	brakes := BrakeModel{MaxBrakeForce: 9000, ABSResponse: 1.5}

	if got := brakes.LockupRisk(0, 1000); got != 0 {
		t.Fatalf("expected zero risk with no brake input, got %.2f", got)
	}

	if got := brakes.LockupRisk(0.9, 30000); got >= 0.5 {
		t.Fatalf("expected low risk with high grip, got %.2f", got)
	}

	if got := brakes.LockupRisk(1, 1000); got != 1 {
		t.Fatalf("expected full risk with low grip, got %.2f", got)
	}
}

func TestLateralGripFromSpeed(t *testing.T) {
	curve := TireGripCurve{PeakSlip: 0.1, PeakGrip: 1.2, SlideGrip: 0.8}
	aero := AeroModel{BaseDownforce: 500, DownforcePerMS2: 2, MaxDownforce: 0}

	gripLow := LateralGripFromSpeed(curve, aero, 10)
	gripHigh := LateralGripFromSpeed(curve, aero, 40)

	if gripHigh <= gripLow {
		t.Fatalf("expected more grip at higher speed")
	}
}
