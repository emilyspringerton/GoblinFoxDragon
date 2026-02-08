package main

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

const (
	W = 44
	H = 44

	NUM_BOIDS   = 140
	NUM_GOBLINS = 48
	NUM_FOXES   = 32
	DECAY       = 0.96

	DragonIntervalTicks = 2000
	DragonDurationTicks = 150
)

type Vec2 struct {
	X, Y float64
}

func (v Vec2) Add(o Vec2) Vec2 { return Vec2{v.X + o.X, v.Y + o.Y} }
func (v Vec2) Sub(o Vec2) Vec2 { return Vec2{v.X - o.X, v.Y - o.Y} }
func (v Vec2) Mul(f float64) Vec2 {
	return Vec2{v.X * f, v.Y * f}
}
func (v Vec2) Len() float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y) }

func (v Vec2) Normalize() Vec2 {
	l := v.Len()
	if l == 0 {
		return Vec2{}
	}
	return v.Mul(1 / l)
}

type Boid struct {
	Pos Vec2
	Vel Vec2

	Speed float64
	Aggro float64
	Eff   float64
}

type GoblinType uint8

const (
	GoblinScavenger GoblinType = iota
	GoblinTinkerer
	GoblinRaider
	GoblinMerchant
)

type Goblin struct {
	X, Y   int
	Kind   GoblinType
	Energy float64
	Aggro  float64
	Greed  float64
}

type FoxType uint8

const (
	FoxCourier FoxType = iota
	FoxPitFix
	FoxApex
	FoxShow
)

type Fox struct {
	X, Y   int
	VX, VY int
	Kind   FoxType
	Energy float64
}

var boids []Boid
var goblins []Goblin
var foxes []Fox

var pheromone [H][W]float64
var power [H][W]float64
var city [H][W]float64
var entropy [H][W]float64
var rubble [H][W]float64
var market [H][W]float64
var lane [H][W]float64
var security [H][W]float64
var spotlight [H][W]float64
var dragonHeat [H][W]float64

var dragonActive bool
var dragonTick int
var dragonX int
var dragonY int

func clear() {
	fmt.Print("\x1b[2J\x1b[H")
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func dist(a, b Vec2) float64 {
	return a.Sub(b).Len()
}

func initWorld() {
	for i := 0; i < NUM_BOIDS; i++ {
		boids = append(boids, Boid{
			Pos:   Vec2{rand.Float64() * W, rand.Float64() * H},
			Vel:   Vec2{rand.Float64()*2 - 1, rand.Float64()*2 - 1},
			Speed: 0.5 + rand.Float64(),
			Aggro: rand.Float64(),
			Eff:   rand.Float64(),
		})
	}

	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			city[y][x] = rand.Float64() * 0.3
			entropy[y][x] = rand.Float64() * 0.1
			rubble[y][x] = rand.Float64() * 0.05
			lane[y][x] = rand.Float64() * 0.05
			security[y][x] = rand.Float64() * 0.05
		}
	}

	for i := 0; i < NUM_GOBLINS; i++ {
		goblins = append(goblins, Goblin{
			X:      rand.Intn(W),
			Y:      rand.Intn(H),
			Kind:   GoblinType(rand.Intn(4)),
			Energy: 0.5 + rand.Float64(),
			Aggro:  rand.Float64(),
			Greed:  rand.Float64(),
		})
	}

	for i := 0; i < NUM_FOXES; i++ {
		foxes = append(foxes, Fox{
			X:      rand.Intn(W),
			Y:      rand.Intn(H),
			VX:     rand.Intn(3) - 1,
			VY:     rand.Intn(3) - 1,
			Kind:   FoxType(rand.Intn(4)),
			Energy: 0.5 + rand.Float64(),
		})
	}
}

func boidForces(b *Boid) Vec2 {
	var align Vec2
	var coh Vec2
	var sep Vec2

	count := 0

	for i := range boids {
		o := &boids[i]
		d := dist(b.Pos, o.Pos)

		if d > 0 && d < 6 {
			align = align.Add(o.Vel)
			coh = coh.Add(o.Pos)

			if d < 2 {
				sep = sep.Add(b.Pos.Sub(o.Pos))
			}

			count++
		}
	}

	if count > 0 {
		align = align.Mul(1 / float64(count)).Normalize()
		coh = coh.Mul(1 / float64(count)).Sub(b.Pos).Normalize()
		sep = sep.Normalize()
	}

	return align.Add(coh).Add(sep.Mul(1.5))
}

func updateBoids() {
	for i := range boids {
		b := &boids[i]

		f := boidForces(b)

		px := int(b.Pos.X)
		py := int(b.Pos.Y)

		if px >= 0 && px < W && py >= 0 && py < H {
			ph := pheromone[py][px]
			if ph > 0.01 {
				f = f.Add(Vec2{rand.Float64()*2 - 1, rand.Float64()*2 - 1}.Mul(ph))
			}
		}

		b.Vel = b.Vel.Add(f).Normalize().Mul(b.Speed)

		b.Pos = b.Pos.Add(b.Vel)

		if b.Pos.X < 0 {
			b.Pos.X += W
		}
		if b.Pos.Y < 0 {
			b.Pos.Y += H
		}
		if b.Pos.X >= W {
			b.Pos.X -= W
		}
		if b.Pos.Y >= H {
			b.Pos.Y -= H
		}

		x := int(b.Pos.X)
		y := int(b.Pos.Y)

		if x >= 0 && x < W && y >= 0 && y < H {
			pheromone[y][x] += 0.4 * b.Eff
			power[y][x] += 0.2
			city[y][x] += 0.02
		}

		if rand.Float64() < 0.001 {
			b.Speed = clamp(b.Speed+rand.NormFloat64()*0.05, 0.2, 2)
			b.Aggro = clamp(b.Aggro+rand.NormFloat64()*0.05, 0, 1)
			b.Eff = clamp(b.Eff+rand.NormFloat64()*0.05, 0, 1)
		}
	}
}

func updateFields() {
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			pheromone[y][x] *= DECAY
			entropy[y][x] *= 0.98
			rubble[y][x] *= 0.985
			market[y][x] *= 0.96
			lane[y][x] *= 0.97
			security[y][x] *= 0.975
			spotlight[y][x] *= 0.92
			dragonHeat[y][x] *= 0.9

			if power[y][x] > 0.1 {
				for dy := -1; dy <= 1; dy++ {
					for dx := -1; dx <= 1; dx++ {
						ny := y + dy
						nx := x + dx
						if nx >= 0 && nx < W && ny >= 0 && ny < H {
							power[ny][nx] += power[y][x] * 0.15
						}
					}
				}
			}

			power[y][x] *= 0.88

			if city[y][x] > 0.3 {
				city[y][x] += 0.01
			} else {
				city[y][x] *= 0.995
			}

			city[y][x] = clamp(city[y][x], 0, 1)
			entropy[y][x] = clamp(entropy[y][x], 0, 2)
			rubble[y][x] = clamp(rubble[y][x], 0, 2)
			market[y][x] = clamp(market[y][x], 0, 2)
			lane[y][x] = clamp(lane[y][x], 0, 2)
			security[y][x] = clamp(security[y][x], 0, 2)
			spotlight[y][x] = clamp(spotlight[y][x], 0, 2)
			dragonHeat[y][x] = clamp(dragonHeat[y][x], 0, 2)
		}
	}
}

func goblinSymbol(kind GoblinType) string {
	switch kind {
	case GoblinScavenger:
		return "g"
	case GoblinTinkerer:
		return "⚙"
	case GoblinRaider:
		return "⚔"
	case GoblinMerchant:
		return "$"
	default:
		return "g"
	}
}

func goblinScore(kind GoblinType, x, y int) float64 {
	trade := pheromone[y][x]
	powerField := power[y][x]
	ent := entropy[y][x]
	rub := rubble[y][x]
	mkt := market[y][x]

	switch kind {
	case GoblinScavenger:
		return rub*2 + ent*1.2 - powerField*0.2
	case GoblinTinkerer:
		return (1-powerField)*1.5 + ent*0.6
	case GoblinRaider:
		return trade*2 - powerField*0.4 + ent
	case GoblinMerchant:
		return ent*1.5 + rub + mkt*0.3
	default:
		return ent + rub
	}
}

func updateGoblins() {
	for i := range goblins {
		g := &goblins[i]
		bestX, bestY := g.X, g.Y
		bestScore := goblinScore(g.Kind, g.X, g.Y)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				nx := (g.X + dx + W) % W
				ny := (g.Y + dy + H) % H
				score := goblinScore(g.Kind, nx, ny) + rand.Float64()*0.05
				if score > bestScore {
					bestScore = score
					bestX, bestY = nx, ny
				}
			}
		}
		g.X, g.Y = bestX, bestY
		g.Energy -= 0.001

		switch g.Kind {
		case GoblinScavenger:
			if rubble[g.Y][g.X] > 0.01 {
				rubble[g.Y][g.X] *= 0.9
				entropy[g.Y][g.X] += 0.05
			}
		case GoblinTinkerer:
			if power[g.Y][g.X] < 0.5 {
				mod := rand.Float64()
				if mod < 0.6 {
					power[g.Y][g.X] += 0.05
					entropy[g.Y][g.X] += 0.08
				} else if mod < 0.9 {
					power[g.Y][g.X] += 0.1
				} else {
					pheromone[g.Y][g.X] += 0.2
				}
			}
		case GoblinRaider:
			if pheromone[g.Y][g.X] > 0.2 {
				pheromone[g.Y][g.X] *= 0.7
				entropy[g.Y][g.X] += 0.2
				rubble[g.Y][g.X] += 0.05
			}
		case GoblinMerchant:
			market[g.Y][g.X] += 0.08
			pheromone[g.Y][g.X] += 0.05
			entropy[g.Y][g.X] += 0.03
		}

		if g.Energy <= 0 {
			g.X = rand.Intn(W)
			g.Y = rand.Intn(H)
			g.Energy = 0.5 + rand.Float64()
		}
	}
}

func foxSymbol(kind FoxType) string {
	switch kind {
	case FoxCourier:
		return "f"
	case FoxPitFix:
		return "🔧"
	case FoxApex:
		return "🏁"
	case FoxShow:
		return "😈"
	default:
		return "f"
	}
}

func foxScore(kind FoxType, x, y int) float64 {
	trade := pheromone[y][x]
	ln := lane[y][x]
	ent := entropy[y][x]
	sec := security[y][x]
	sp := spotlight[y][x]
	pw := power[y][x]

	switch kind {
	case FoxCourier:
		return trade*1.2 + ln*1.8 - ent*0.6
	case FoxPitFix:
		return (1-pw)*1.4 + ent*0.4
	case FoxApex:
		return sec*1.2 + ln*0.8 - ent*0.4
	case FoxShow:
		return sp*2 + trade*0.4
	default:
		return ln + trade
	}
}

func updateFoxes() {
	for i := range foxes {
		f := &foxes[i]
		bestX, bestY := f.X, f.Y
		bestScore := foxScore(f.Kind, f.X, f.Y)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				nx := (f.X + dx + W) % W
				ny := (f.Y + dy + H) % H
				score := foxScore(f.Kind, nx, ny) + rand.Float64()*0.05
				if score > bestScore {
					bestScore = score
					bestX, bestY = nx, ny
				}
			}
		}
		f.VX = bestX - f.X
		f.VY = bestY - f.Y
		f.X, f.Y = bestX, bestY
		f.Energy -= 0.001

		switch f.Kind {
		case FoxCourier:
			lane[f.Y][f.X] += 0.12
			if rubble[f.Y][f.X] > 0.02 {
				rubble[f.Y][f.X] *= 0.85
			}
			pheromone[f.Y][f.X] += 0.04
		case FoxPitFix:
			if entropy[f.Y][f.X] > 0.02 {
				entropy[f.Y][f.X] *= 0.85
			}
			if power[f.Y][f.X] < 1 {
				power[f.Y][f.X] += 0.15
			}
		case FoxApex:
			security[f.Y][f.X] += 0.12
			lane[f.Y][f.X] += 0.05
		case FoxShow:
			spotlight[f.Y][f.X] += 0.2
			pheromone[f.Y][f.X] += 0.05
		}

		if f.Energy <= 0 {
			f.X = rand.Intn(W)
			f.Y = rand.Intn(H)
			f.Energy = 0.5 + rand.Float64()
		}
	}
}

func updateDragon() {
	dragonTick++
	if !dragonActive && dragonTick%DragonIntervalTicks == 0 {
		dragonActive = true
		dragonX = rand.Intn(W)
		dragonY = rand.Intn(H)
	}

	if dragonActive {
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				nx := (dragonX + dx + W) % W
				ny := (dragonY + dy + H) % H
				dragonHeat[ny][nx] += 0.2
				entropy[ny][nx] += 0.1
				security[ny][nx] *= 0.9
				lane[ny][nx] *= 0.9
				spotlight[ny][nx] += 0.15
			}
		}
		if dragonTick%DragonDurationTicks == 0 {
			dragonActive = false
		}
	}
}

func symbol(x, y int) string {
	p := pheromone[y][x]
	pw := power[y][x]
	c := city[y][x]
	e := entropy[y][x]
	r := rubble[y][x]
	m := market[y][x]
	ln := lane[y][x]
	sec := security[y][x]
	sp := spotlight[y][x]
	dh := dragonHeat[y][x]

	switch {
	case dh > 0.8:
		return "🐉"
	case c > 0.85:
		return "■"
	case c > 0.6:
		return "◆"
	case pw > 1.5:
		return "⚡"
	case sp > 0.8:
		return "✹"
	case sec > 0.9:
		return "▣"
	case ln > 0.9:
		return "═"
	case m > 0.6:
		return "$"
	case r > 0.8:
		return "⬣"
	case e > 0.9:
		return "✺"
	case p > 1.2:
		return "●"
	case p > 0.6:
		return "◉"
	case c > 0.2:
		return "❖"
	default:
		return "≈"
	}
}

func render() {
	grid := make([][]string, H)
	for i := range grid {
		grid[i] = make([]string, W)
	}

	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			grid[y][x] = symbol(x, y)
		}
	}

	for i := range boids {
		x := int(boids[i].Pos.X)
		y := int(boids[i].Pos.Y)
		if x >= 0 && x < W && y >= 0 && y < H {
			grid[y][x] = "✦"
		}
	}
	for i := range goblins {
		x := goblins[i].X
		y := goblins[i].Y
		if x >= 0 && x < W && y >= 0 && y < H {
			grid[y][x] = goblinSymbol(goblins[i].Kind)
		}
	}
	for i := range foxes {
		x := foxes[i].X
		y := foxes[i].Y
		if x >= 0 && x < W && y >= 0 && y < H {
			grid[y][x] = foxSymbol(foxes[i].Kind)
		}
	}

	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			fmt.Print(grid[y][x])
		}
		fmt.Println()
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	initWorld()

	for {
		clear()

		updateBoids()
		updateGoblins()
		updateFoxes()
		updateDragon()
		updateFields()

		render()

		time.Sleep(60 * time.Millisecond)
	}
}
