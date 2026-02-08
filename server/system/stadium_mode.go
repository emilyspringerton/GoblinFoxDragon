package system

import "math"

type StadiumTeam int

const (
	TeamBlue StadiumTeam = iota
	TeamOrange
)

type StadiumModeConfig struct {
	FieldLength float64
	FieldWidth  float64
	GoalDepth   float64
	GoalWidth   float64
	GoalHeight  float64
}

type StadiumBall struct {
	Position Vec3
	Velocity Vec3
}

type StadiumScore struct {
	Blue   int
	Orange int
}

type StadiumState struct {
	Ball  StadiumBall
	Score StadiumScore
}

func (c StadiumModeConfig) GoalCenterFor(team StadiumTeam) Vec3 {
	sign := 1.0
	if team == TeamBlue {
		sign = -1
	}
	return Vec3{X: 0, Y: c.GoalHeight * 0.5, Z: sign * (c.FieldLength*0.5 + c.GoalDepth*0.5)}
}

func DetectGoal(ball StadiumBall, cfg StadiumModeConfig) (StadiumTeam, bool) {
	if cfg.FieldLength <= 0 || cfg.FieldWidth <= 0 {
		return TeamBlue, false
	}

	if math.Abs(ball.Position.X) > cfg.GoalWidth*0.5 {
		return TeamBlue, false
	}
	if ball.Position.Y < 0 || ball.Position.Y > cfg.GoalHeight {
		return TeamBlue, false
	}

	goalLine := cfg.FieldLength * 0.5
	if ball.Position.Z > goalLine {
		return TeamBlue, true
	}
	if ball.Position.Z < -goalLine {
		return TeamOrange, true
	}
	return TeamBlue, false
}

func (s StadiumState) WithGoal(team StadiumTeam) StadiumState {
	if team == TeamBlue {
		s.Score.Blue++
	} else {
		s.Score.Orange++
	}
	return s
}

func (s StadiumState) ResetBall(cfg StadiumModeConfig) StadiumState {
	s.Ball.Position = Vec3{X: 0, Y: cfg.GoalHeight * 0.5, Z: 0}
	s.Ball.Velocity = Vec3{}
	return s
}
