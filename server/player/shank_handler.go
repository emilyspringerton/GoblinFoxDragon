package player

import (
	"dragonsnshit/server/system"
)

type DamageCause int

const (
	CauseProjectile DamageCause = iota
)

type DamageSource struct {
	Cause DamageCause
}

type LivingEntity interface {
	Hurt(amount float64, source DamageSource)
}

type RaycastResult interface {
	Position() system.Vec3
}

type EntityResult interface {
	RaycastResult
	Entity() LivingEntity
}

type RaycastWorld interface {
	RayTrace(start, end system.Vec3) (RaycastResult, bool)
}

type ShankPlayer interface {
	Position() system.Vec3
	EyeHeight() float64
	World() RaycastWorld
	SendSound(name string, pos system.Vec3)
}

func HandleShankFire(p ShankPlayer, yaw, pitch float64, weaponID int) (bool, system.Vec3, bool) {
	dir := system.DirectionFromYawPitch(yaw, pitch)

	dist := system.DefaultRange
	if weaponID == 1 {
		dist = system.MagnumRange
	}

	start := p.Position().Add(system.Vec3{X: 0, Y: p.EyeHeight(), Z: 0})
	end := start.Add(dir.Mul(dist))

	res, found := p.World().RayTrace(start, end)
	if !found {
		return false, system.Vec3{}, false
	}

	pos := res.Position()
	if entityResult, ok := res.(EntityResult); ok {
		e := entityResult.Entity()
		e.Hurt(50.0, DamageSource{Cause: CauseProjectile})
		p.SendSound("random.orb", pos)
		return true, pos, true
	}

	p.SendSound("step.wood", pos)
	return true, pos, false
}
