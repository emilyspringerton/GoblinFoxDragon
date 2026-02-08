package system

import "testing"

func TestStepVehicleAccelerates(t *testing.T) {
	state := VehicleState{}
	cfg := VehicleConfig{Mass: 800, MaxEngineForce: 12000, MaxBrakeForce: 9000, DragCoefficient: 0.35, RollingResistance: 12, Wheelbase: 3.2}
	tire := TireGripCurve{PeakSlip: 0.12, PeakGrip: 1.4, SlideGrip: 0.9}
	aero := AeroModel{BaseDownforce: 600, DownforcePerMS2: 2.5}
	brakes := BrakeModel{MaxBrakeForce: cfg.MaxBrakeForce, ABSResponse: 1.3}

	updated, telemetry := StepVehicle(state, VehicleInput{Throttle: 1}, cfg, tire, aero, brakes, 0.1)
	if updated.Velocity.X == 0 && updated.Velocity.Z == 0 {
		t.Fatalf("expected velocity to increase")
	}
	if telemetry.Speed != 0 {
		t.Fatalf("expected initial telemetry speed to be zero, got %.3f", telemetry.Speed)
	}
}

func TestStepVehicleBrakes(t *testing.T) {
	state := VehicleState{Velocity: Vec3{X: 30}}
	cfg := VehicleConfig{Mass: 900, MaxEngineForce: 11000, MaxBrakeForce: 12000, DragCoefficient: 0.25, RollingResistance: 10, Wheelbase: 3.4, SurfaceGrip: 1}
	tire := TireGripCurve{PeakSlip: 0.1, PeakGrip: 1.3, SlideGrip: 0.85}
	aero := AeroModel{BaseDownforce: 800, DownforcePerMS2: 3.0}
	brakes := BrakeModel{MaxBrakeForce: cfg.MaxBrakeForce, ABSResponse: 1.5}

	updated, telemetry := StepVehicle(state, VehicleInput{Brake: 1}, cfg, tire, aero, brakes, 0.1)
	if updated.Velocity.X >= state.Velocity.X {
		t.Fatalf("expected braking to reduce speed")
	}
	if telemetry.Lockup <= 0 {
		t.Fatalf("expected some lockup risk")
	}
}

func TestStepVehicleSteers(t *testing.T) {
	state := VehicleState{Velocity: Vec3{X: 40}}
	cfg := VehicleConfig{Mass: 820, MaxEngineForce: 0, MaxBrakeForce: 0, DragCoefficient: 0.2, RollingResistance: 6, Wheelbase: 3.0, Steering: SteeringModel{LowSpeedLimit: 1, HighSpeedLimit: 1, TransitionSpeed: 1}, SurfaceGrip: 1}
	tire := TireGripCurve{PeakSlip: 0.11, PeakGrip: 1.5, SlideGrip: 1.0}
	aero := AeroModel{BaseDownforce: 900, DownforcePerMS2: 3.2}
	brakes := BrakeModel{MaxBrakeForce: cfg.MaxBrakeForce, ABSResponse: 1.2}

	updated, _ := StepVehicle(state, VehicleInput{Steer: 0.4}, cfg, tire, aero, brakes, 0.1)
	if updated.Yaw == state.Yaw {
		t.Fatalf("expected yaw to change with steering input")
	}
}

func TestStepVehicleConservesMomentumWithoutForces(t *testing.T) {
	state := VehicleState{Velocity: Vec3{X: 20, Z: -5}}
	cfg := VehicleConfig{Mass: 800, MaxEngineForce: 0, MaxBrakeForce: 0, DragCoefficient: 0, RollingResistance: 0, Wheelbase: 3.1, SurfaceGrip: 0}
	tire := TireGripCurve{PeakSlip: 0.1, PeakGrip: 0, SlideGrip: 0}
	aero := AeroModel{BaseDownforce: 0, DownforcePerMS2: 0}
	brakes := BrakeModel{MaxBrakeForce: 0, ABSResponse: 1.0}

	updated, _ := StepVehicle(state, VehicleInput{}, cfg, tire, aero, brakes, 0.1)
	if updated.Velocity != state.Velocity {
		t.Fatalf("expected velocity to remain constant without forces")
	}
}

func TestSteeringModelAppliesSpeedLimit(t *testing.T) {
	model := SteeringModel{LowSpeedLimit: 1, HighSpeedLimit: 0.4, TransitionSpeed: 50}

	lowSpeed := model.Apply(1, 5)
	highSpeed := model.Apply(1, 100)

	if lowSpeed <= highSpeed {
		t.Fatalf("expected low-speed steering to be less restricted than high-speed steering")
	}
	if highSpeed != 0.4 {
		t.Fatalf("expected high-speed steering to be clamped to 0.4, got %.2f", highSpeed)
	}
}

func TestSurfaceGripAffectsLockupRisk(t *testing.T) {
	state := VehicleState{Velocity: Vec3{X: 25}}
	cfg := VehicleConfig{Mass: 900, MaxEngineForce: 0, MaxBrakeForce: 12000, DragCoefficient: 0.1, RollingResistance: 5, Wheelbase: 3.4, SurfaceGrip: 1}
	tire := TireGripCurve{PeakSlip: 0.1, PeakGrip: 1.3, SlideGrip: 0.85}
	aero := AeroModel{BaseDownforce: 800, DownforcePerMS2: 3.0}
	brakes := BrakeModel{MaxBrakeForce: cfg.MaxBrakeForce, ABSResponse: 1.5}

	_, highGripTelemetry := StepVehicle(state, VehicleInput{Brake: 1}, cfg, tire, aero, brakes, 0.1)

	cfg.SurfaceGrip = 0.3
	_, lowGripTelemetry := StepVehicle(state, VehicleInput{Brake: 1}, cfg, tire, aero, brakes, 0.1)

	if lowGripTelemetry.Lockup <= highGripTelemetry.Lockup {
		t.Fatalf("expected lower grip to raise lockup risk")
	}
}
