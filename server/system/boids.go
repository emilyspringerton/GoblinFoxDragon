package system

import "math"

type BoidState struct {
	Pos Vec3
	Vel Vec3
}

type BoidConfig struct {
	NeighborRadius   float64
	SeparationRadius float64
	AlignmentWeight  float64
	CohesionWeight   float64
	SeparationWeight float64
	MaxSpeed         float64
}

func StepBoids(boids []BoidState, cfg BoidConfig) []BoidState {
	if len(boids) == 0 {
		return nil
	}
	updated := make([]BoidState, len(boids))

	for i := range boids {
		b := boids[i]
		var align Vec3
		var cohesion Vec3
		var separation Vec3
		count := 0

		for j := range boids {
			if i == j {
				continue
			}
			o := boids[j]
			delta := vec3Sub(o.Pos, b.Pos)
			d := vec3Len(delta)
			if d <= 0 || d > cfg.NeighborRadius {
				continue
			}
			align = align.Add(o.Vel)
			cohesion = cohesion.Add(o.Pos)
			if d < cfg.SeparationRadius {
				separation = separation.Add(vec3Sub(b.Pos, o.Pos))
			}
			count++
		}

		if count > 0 {
			inv := 1.0 / float64(count)
			align = vec3Normalize(align.Mul(inv))
			cohesion = vec3Normalize(vec3Sub(cohesion.Mul(inv), b.Pos))
			separation = vec3Normalize(separation)
		}

		force := align.Mul(cfg.AlignmentWeight).
			Add(cohesion.Mul(cfg.CohesionWeight)).
			Add(separation.Mul(cfg.SeparationWeight))

		newVel := b.Vel.Add(force)
		if cfg.MaxSpeed > 0 {
			if vec3Len(newVel) == 0 && vec3Len(force) > 0 {
				newVel = vec3Normalize(force).Mul(cfg.MaxSpeed)
			} else {
				newVel = vec3Normalize(newVel).Mul(cfg.MaxSpeed)
			}
		}
		updated[i] = BoidState{Pos: b.Pos.Add(newVel), Vel: newVel}
	}

	return updated
}

func vec3Sub(a, b Vec3) Vec3 {
	return Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}

func vec3Len(v Vec3) float64 {
	return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z)
}

func vec3Normalize(v Vec3) Vec3 {
	l := vec3Len(v)
	if l == 0 {
		return Vec3{}
	}
	return v.Mul(1 / l)
}
