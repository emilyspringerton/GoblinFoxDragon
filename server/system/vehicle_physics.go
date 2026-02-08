package system

import "math"

type TireGripCurve struct {
	PeakSlip  float64
	PeakGrip  float64
	SlideGrip float64
}

func (t TireGripCurve) GripForSlip(slipAngle float64) float64 {
	absSlip := math.Abs(slipAngle)
	if absSlip <= 0 {
		return 0
	}
	if absSlip <= t.PeakSlip {
		return t.PeakGrip * (absSlip / t.PeakSlip)
	}

	falloffSlip := t.PeakSlip * 2
	if absSlip >= falloffSlip {
		return t.SlideGrip
	}

	falloffRange := falloffSlip - t.PeakSlip
	slipPastPeak := absSlip - t.PeakSlip
	return t.PeakGrip - (t.PeakGrip-t.SlideGrip)*(slipPastPeak/falloffRange)
}

type AeroModel struct {
	BaseDownforce   float64
	DownforcePerMS2 float64
	MaxDownforce    float64
}

func (a AeroModel) Downforce(speed float64) float64 {
	if speed <= 0 {
		return a.BaseDownforce
	}
	force := a.BaseDownforce + a.DownforcePerMS2*(speed*speed)
	if a.MaxDownforce > 0 && force > a.MaxDownforce {
		return a.MaxDownforce
	}
	return force
}

type BrakeModel struct {
	MaxBrakeForce float64
	ABSResponse   float64
}

func (b BrakeModel) LockupRisk(brakeInput, availableGrip float64) float64 {
	if brakeInput <= 0 || b.MaxBrakeForce <= 0 {
		return 0
	}

	reqForce := b.MaxBrakeForce * brakeInput
	if availableGrip <= 0 {
		return 1
	}

	risk := reqForce / availableGrip
	if b.ABSResponse > 0 {
		risk = math.Pow(risk, 1/b.ABSResponse)
	}

	if risk < 0 {
		return 0
	}
	if risk > 1 {
		return 1
	}
	return risk
}

func LateralGripFromSpeed(curve TireGripCurve, aero AeroModel, speed float64) float64 {
	downforce := aero.Downforce(speed)
	return curve.PeakGrip + downforce
}
