package system

import "math"

type VehicleState struct {
	Position Vec3
	Velocity Vec3
	Yaw      float64
}

type VehicleInput struct {
	Throttle float64
	Brake    float64
	Steer    float64
}

type VehicleConfig struct {
	Mass              float64
	MaxEngineForce    float64
	MaxBrakeForce     float64
	DragCoefficient   float64
	RollingResistance float64
	Wheelbase         float64
	Steering          SteeringModel
	SurfaceGrip       float64
}

type SteeringModel struct {
	LowSpeedLimit   float64
	HighSpeedLimit  float64
	TransitionSpeed float64
}

func (s SteeringModel) Apply(input, speed float64) float64 {
	if s.TransitionSpeed <= 0 || (s.LowSpeedLimit == 0 && s.HighSpeedLimit == 0) {
		return input
	}

	t := math.Min(speed/s.TransitionSpeed, 1)
	limit := s.LowSpeedLimit + (s.HighSpeedLimit-s.LowSpeedLimit)*t
	if limit < 0 {
		limit = 0
	}
	if limit > 1 {
		limit = 1
	}
	if input > limit {
		return limit
	}
	if input < -limit {
		return -limit
	}
	return input
}

type VehicleTelemetry struct {
	Speed     float64
	SlipAngle float64
	Lockup    float64
}

func StepVehicle(state VehicleState, input VehicleInput, cfg VehicleConfig, tire TireGripCurve, aero AeroModel, brakes BrakeModel, dt float64) (VehicleState, VehicleTelemetry) {
	clamp01 := func(v float64) float64 {
		if v < 0 {
			return 0
		}
		if v > 1 {
			return 1
		}
		return v
	}

	throttle := clamp01(input.Throttle)
	brake := clamp01(input.Brake)

	forward := Vec3{X: math.Cos(state.Yaw), Y: 0, Z: math.Sin(state.Yaw)}
	right := Vec3{X: -math.Sin(state.Yaw), Y: 0, Z: math.Cos(state.Yaw)}

	longitudinal := state.Velocity.X*forward.X + state.Velocity.Z*forward.Z
	lateral := state.Velocity.X*right.X + state.Velocity.Z*right.Z

	speed := math.Hypot(longitudinal, lateral)
	var slipAngle float64
	if speed > 0.001 {
		slipAngle = math.Atan2(lateral, math.Abs(longitudinal))
	}

	engineForce := cfg.MaxEngineForce * throttle
	gripScale := cfg.SurfaceGrip
	if gripScale <= 0 {
		gripScale = 1
	}
	availableGrip := LateralGripFromSpeed(tire, aero, speed) * cfg.Mass * 9.81 * gripScale
	lockupRisk := brakes.LockupRisk(brake, availableGrip)
	brakeForce := cfg.MaxBrakeForce * brake * (1 - lockupRisk)

	dragForce := cfg.DragCoefficient * speed * speed
	rollForce := cfg.RollingResistance * speed

	longitudinalForce := engineForce - brakeForce - dragForce - rollForce
	if longitudinal < 0 {
		longitudinalForce = engineForce + brakeForce - dragForce - rollForce
	}

	maxLongForce := availableGrip
	if maxLongForce > 0 {
		longitudinalForce = math.Copysign(math.Min(math.Abs(longitudinalForce), maxLongForce), longitudinalForce)
	}

	maxLatForce := tire.GripForSlip(slipAngle) * cfg.Mass * 9.81 * gripScale
	remainingGrip := math.Sqrt(math.Max(availableGrip*availableGrip-longitudinalForce*longitudinalForce, 0))
	latForce := -math.Copysign(math.Min(math.Abs(maxLatForce), remainingGrip), lateral)

	accelLong := longitudinalForce / cfg.Mass
	accelLat := latForce / cfg.Mass

	accelWorld := forward.Mul(accelLong).Add(right.Mul(accelLat))
	state.Velocity = state.Velocity.Add(accelWorld.Mul(dt))
	state.Position = state.Position.Add(state.Velocity.Mul(dt))

	if cfg.Wheelbase > 0 {
		steer := cfg.Steering.Apply(input.Steer, speed)
		yawRate := (speed / cfg.Wheelbase) * steer
		state.Yaw += yawRate * dt
	}

	return state, VehicleTelemetry{
		Speed:     speed,
		SlipAngle: slipAngle,
		Lockup:    lockupRisk,
	}
}
