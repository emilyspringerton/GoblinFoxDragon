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

// Sub/Dot/Len (backend-unification, 2026-08-03, real entity hit detection in apps2/server-go --
// ported from SHANKPIT's own sibling repo's real, tested gameWorld.RayTrace, which already
// relies on exactly these three operations for ray-vs-player-sphere intersection). Not present
// in this repo's own Vec3 before now -- GoblinFoxDragon's copy only ever needed Add/Mul until
// this.
func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{X: v.X - o.X, Y: v.Y - o.Y, Z: v.Z - o.Z}
}

func (v Vec3) Dot(o Vec3) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

func (v Vec3) Len() float64 {
	return math.Sqrt(v.Dot(v))
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
