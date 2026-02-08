package system

import "math"

const (
	DefaultRange = 100.0
	MagnumRange  = 200.0
)

type Vec3 struct {
	X float64
	Y float64
	Z float64
}

func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{X: v.X + o.X, Y: v.Y + o.Y, Z: v.Z + o.Z}
}

func (v Vec3) Mul(scale float64) Vec3 {
	return Vec3{X: v.X * scale, Y: v.Y * scale, Z: v.Z * scale}
}

func DirectionFromYawPitch(yaw, pitch float64) Vec3 {
	yRad := (yaw + 90) * (math.Pi / 180)
	pRad := pitch * (math.Pi / 180)

	return Vec3{
		X: math.Cos(yRad) * math.Cos(pRad),
		Y: math.Sin(pRad),
		Z: math.Sin(yRad) * math.Cos(pRad),
	}
}
