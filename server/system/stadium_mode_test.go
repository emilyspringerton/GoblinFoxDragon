package system

import "testing"

func TestDetectGoal(t *testing.T) {
	cfg := StadiumModeConfig{FieldLength: 120, FieldWidth: 90, GoalDepth: 10, GoalWidth: 30, GoalHeight: 12}

	ball := StadiumBall{Position: Vec3{X: 0, Y: 6, Z: 61}}
	team, scored := DetectGoal(ball, cfg)
	if !scored || team != TeamBlue {
		t.Fatalf("expected blue goal to score")
	}

	ball = StadiumBall{Position: Vec3{X: 0, Y: 6, Z: -61}}
	team, scored = DetectGoal(ball, cfg)
	if !scored || team != TeamOrange {
		t.Fatalf("expected orange goal to score")
	}
}

func TestDetectGoalOutOfBounds(t *testing.T) {
	cfg := StadiumModeConfig{FieldLength: 120, FieldWidth: 90, GoalDepth: 10, GoalWidth: 30, GoalHeight: 12}

	ball := StadiumBall{Position: Vec3{X: 20, Y: 6, Z: 61}}
	_, scored := DetectGoal(ball, cfg)
	if scored {
		t.Fatalf("expected no goal when ball is outside goal width")
	}

	ball = StadiumBall{Position: Vec3{X: 0, Y: 14, Z: -61}}
	_, scored = DetectGoal(ball, cfg)
	if scored {
		t.Fatalf("expected no goal when ball is above goal height")
	}
}

func TestResetBall(t *testing.T) {
	cfg := StadiumModeConfig{FieldLength: 120, FieldWidth: 90, GoalDepth: 10, GoalWidth: 30, GoalHeight: 12}
	state := StadiumState{Ball: StadiumBall{Position: Vec3{X: 5, Y: 2, Z: 3}, Velocity: Vec3{X: 1, Y: 0, Z: -2}}}

	state = state.ResetBall(cfg)
	if state.Ball.Position != (Vec3{X: 0, Y: 6, Z: 0}) {
		t.Fatalf("expected ball position to reset to center")
	}
	if state.Ball.Velocity != (Vec3{}) {
		t.Fatalf("expected ball velocity to reset")
	}
}

func TestWithGoalUpdatesScore(t *testing.T) {
	state := StadiumState{}
	state = state.WithGoal(TeamBlue)
	state = state.WithGoal(TeamOrange)
	state = state.WithGoal(TeamOrange)

	if state.Score.Blue != 1 || state.Score.Orange != 2 {
		t.Fatalf("unexpected score totals")
	}
}
