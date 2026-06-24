// Package beatsync implements the TRAPX BeatSync stub service.
//
// BeatSync synchronises game world events to a music beat grid. The stub
// implementation emits Beat events at a configured BPM without requiring
// real audio input. The real implementation (M2 milestone, pending TRAPX
// stem delivery) will parse an audio stem file to extract a beat grid.
//
// # Beat events
//
// Each Beat carries a BeatType (Kick/Snare/Hat/Bass) that downstream consumers
// use to modulate world state:
//
//   - Kick:  ReactiveSky color pulse (primary beat visual)
//   - Snare: weather phase bias toggle
//   - Bass:  mob swing timer modulation (Bass → Storm bias)
//   - Hat:   ambient tick (minor crowd movement modulation)
//
// The Engine emits Beats on a channel. Callers read from BeatCh; the channel
// is unbuffered so a slow consumer causes the engine to skip beats rather than
// queue them indefinitely.
package beatsync

import (
	"context"
	"math"
	"time"
)

// BeatType identifies which element of the beat grid fired.
type BeatType int

const (
	BeatKick  BeatType = iota // primary downbeat; sky pulse
	BeatSnare                 // backbeat; weather toggle
	BeatBass                  // bass hit; mob swing modulation
	BeatHat                   // hi-hat tick; ambient modulation
)

var beatTypeNames = map[BeatType]string{
	BeatKick:  "KICK",
	BeatSnare: "SNARE",
	BeatBass:  "BASS",
	BeatHat:   "HAT",
}

// BeatTypeName returns the display name for a BeatType.
func BeatTypeName(b BeatType) string {
	if s, ok := beatTypeNames[b]; ok {
		return s
	}
	return "UNKNOWN"
}

// Beat is a single beat event emitted by the Engine.
type Beat struct {
	At       time.Time
	BPM      float64
	Type     BeatType
	Measure  int     // 1-indexed measure number
	Position int     // beat position within the measure (1–4 for 4/4)
	Strength float64 // 0.0–1.0; amplitude for the stub's sine approximation
}

// WorldEffect returns a descriptor of the world effect this Beat should trigger.
func (b Beat) WorldEffect() string {
	switch b.Type {
	case BeatKick:
		return "sky_pulse"
	case BeatSnare:
		return "weather_toggle"
	case BeatBass:
		return "mob_swing_storm"
	case BeatHat:
		return "crowd_ambient"
	}
	return "none"
}

// Engine is the BeatSync service. It emits Beat events at a configured BPM.
// Not goroutine-safe for configuration changes; start it once and read BeatCh.
type Engine struct {
	BPM    float64
	BeatCh chan Beat // caller reads from this channel

	// StemMix is a placeholder for future real audio stem data.
	// In the stub, it is unused. In M2, it will carry the stem file path.
	StemMix string
}

// NewEngine creates a stub BeatSync engine at the given BPM.
// BeatCh is unbuffered; a slow consumer causes skipped beats.
func NewEngine(bpm float64) *Engine {
	if bpm <= 0 {
		bpm = 140.0
	}
	return &Engine{BPM: bpm, BeatCh: make(chan Beat)}
}

// Run starts the beat emission loop. It blocks until ctx is cancelled.
// Beats are emitted at intervals of (60/BPM) seconds.
// The pattern cycles Kick → Hat → Snare → Hat → (repeat) with Bass
// added on beat 1 of every other measure.
func (e *Engine) Run(ctx context.Context) {
	interval := time.Duration(float64(time.Minute) / e.BPM)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	beat := 0
	measure := 1
	position := 1

	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			beat++
			b := Beat{
				At:      t,
				BPM:     e.BPM,
				Measure: measure,
				Position: position,
				Strength: e.sineStrength(beat),
			}
			b.Type = e.patternType(position, measure)
			// Non-blocking send: skip if consumer is behind.
			select {
			case e.BeatCh <- b:
			default:
			}
			position++
			if position > 4 {
				position = 1
				measure++
			}
		}
	}
}

// patternType returns the BeatType for a given position in a 4/4 measure.
// Pattern: 1=Kick(+Bass every other measure), 2=Hat, 3=Snare, 4=Hat.
func (e *Engine) patternType(position, measure int) BeatType {
	switch position {
	case 1:
		if measure%2 == 0 {
			return BeatBass
		}
		return BeatKick
	case 2, 4:
		return BeatHat
	case 3:
		return BeatSnare
	}
	return BeatKick
}

// sineStrength returns a 0.0–1.0 amplitude for the nth beat, approximating
// a musical dynamics curve. Stub only — real implementation reads stem amplitude.
func (e *Engine) sineStrength(beatNum int) float64 {
	// Slow sine over 32 beats gives gentle dynamics variation.
	return 0.5 + 0.5*math.Sin(float64(beatNum)*math.Pi/16)
}

// Tick generates a single Beat synchronously at the given time.
// Useful for testing without goroutines.
func (e *Engine) Tick(beatNum int, now time.Time) Beat {
	measure := (beatNum-1)/4 + 1
	position := (beatNum-1)%4 + 1
	return Beat{
		At:       now,
		BPM:      e.BPM,
		Measure:  measure,
		Position: position,
		Strength: e.sineStrength(beatNum),
		Type:     e.patternType(position, measure),
	}
}
