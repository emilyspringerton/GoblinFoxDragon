package beatsync

import (
	"context"
	"math"
	"testing"
	"time"
)

var t0 = time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)

// ── Engine construction ───────────────────────────────────────────────────────

func TestNewEngineDefaultBPM(t *testing.T) {
	e := NewEngine(0) // zero → default 140
	if e.BPM != 140.0 {
		t.Errorf("expected default BPM 140, got %.1f", e.BPM)
	}
}

func TestNewEngineCustomBPM(t *testing.T) {
	e := NewEngine(90)
	if e.BPM != 90.0 {
		t.Errorf("expected BPM 90, got %.1f", e.BPM)
	}
}

func TestBeatChIsNonNil(t *testing.T) {
	e := NewEngine(140)
	if e.BeatCh == nil {
		t.Error("BeatCh should be non-nil")
	}
}

// ── Tick (sync beat generation) ───────────────────────────────────────────────

func TestTickBeat1IsKick(t *testing.T) {
	e := NewEngine(140)
	b := e.Tick(1, t0)
	if b.Type != BeatKick {
		t.Errorf("beat 1 (measure 1, pos 1): expected KICK, got %s", BeatTypeName(b.Type))
	}
}

func TestTickBeat2IsHat(t *testing.T) {
	e := NewEngine(140)
	b := e.Tick(2, t0)
	if b.Type != BeatHat {
		t.Errorf("beat 2: expected HAT, got %s", BeatTypeName(b.Type))
	}
}

func TestTickBeat3IsSnare(t *testing.T) {
	e := NewEngine(140)
	b := e.Tick(3, t0)
	if b.Type != BeatSnare {
		t.Errorf("beat 3: expected SNARE, got %s", BeatTypeName(b.Type))
	}
}

func TestTickBeat4IsHat(t *testing.T) {
	e := NewEngine(140)
	b := e.Tick(4, t0)
	if b.Type != BeatHat {
		t.Errorf("beat 4: expected HAT, got %s", BeatTypeName(b.Type))
	}
}

func TestTickBeat5IsBass(t *testing.T) {
	// Beat 5 = measure 2, position 1 → Bass (every even measure)
	e := NewEngine(140)
	b := e.Tick(5, t0)
	if b.Type != BeatBass {
		t.Errorf("beat 5 (measure 2, pos 1): expected BASS, got %s", BeatTypeName(b.Type))
	}
}

func TestTickMeasurePosition(t *testing.T) {
	e := NewEngine(140)
	cases := []struct {
		beatNum  int
		measure  int
		position int
	}{
		{1, 1, 1},
		{4, 1, 4},
		{5, 2, 1},
		{8, 2, 4},
		{9, 3, 1},
	}
	for _, tc := range cases {
		b := e.Tick(tc.beatNum, t0)
		if b.Measure != tc.measure || b.Position != tc.position {
			t.Errorf("beat %d: expected measure=%d pos=%d, got measure=%d pos=%d",
				tc.beatNum, tc.measure, tc.position, b.Measure, b.Position)
		}
	}
}

func TestTickStrengthInRange(t *testing.T) {
	e := NewEngine(140)
	for i := 1; i <= 32; i++ {
		b := e.Tick(i, t0)
		if b.Strength < 0 || b.Strength > 1 {
			t.Errorf("beat %d: Strength %.4f out of [0,1]", i, b.Strength)
		}
	}
}

// ── WorldEffect ───────────────────────────────────────────────────────────────

func TestWorldEffectKick(t *testing.T) {
	e := NewEngine(140)
	b := e.Tick(1, t0)
	if b.WorldEffect() != "sky_pulse" {
		t.Errorf("expected sky_pulse for KICK, got %q", b.WorldEffect())
	}
}

func TestWorldEffectSnare(t *testing.T) {
	e := NewEngine(140)
	b := e.Tick(3, t0)
	if b.WorldEffect() != "weather_toggle" {
		t.Errorf("expected weather_toggle for SNARE, got %q", b.WorldEffect())
	}
}

func TestWorldEffectBass(t *testing.T) {
	e := NewEngine(140)
	b := e.Tick(5, t0) // beat 5 = measure 2, pos 1 → BASS
	if b.WorldEffect() != "mob_swing_storm" {
		t.Errorf("expected mob_swing_storm for BASS, got %q", b.WorldEffect())
	}
}

func TestWorldEffectHat(t *testing.T) {
	e := NewEngine(140)
	b := e.Tick(2, t0)
	if b.WorldEffect() != "crowd_ambient" {
		t.Errorf("expected crowd_ambient for HAT, got %q", b.WorldEffect())
	}
}

// ── Run (async beat emission) ─────────────────────────────────────────────────

func TestRunEmitsBeats(t *testing.T) {
	// Use very high BPM so beats arrive in test time.
	e := NewEngine(3000) // 50ms per beat
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	go e.Run(ctx)

	received := 0
	timeout := time.After(300 * time.Millisecond)
loop:
	for {
		select {
		case <-e.BeatCh:
			received++
			if received >= 2 {
				break loop
			}
		case <-timeout:
			break loop
		}
	}
	if received < 2 {
		t.Errorf("expected at least 2 beats, got %d", received)
	}
}

func TestRunStopsOnCancel(t *testing.T) {
	e := NewEngine(3000)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		e.Run(ctx)
		close(done)
	}()
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Error("Run should stop when context is cancelled")
	}
}

// ── Strength curve ────────────────────────────────────────────────────────────

func TestSineStrengthSymmetry(t *testing.T) {
	e := NewEngine(140)
	// At beat 0 and beat 32, the sine cycle completes; strength should be equal.
	s0 := e.sineStrength(0)
	s32 := e.sineStrength(32)
	if math.Abs(s0-s32) > 0.001 {
		t.Errorf("sine strength should be periodic: s0=%.4f s32=%.4f", s0, s32)
	}
}
