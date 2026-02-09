// Source [5]
package parser

import "io"

type ChunkType string

const (
	TypeVision     ChunkType = "VISION"
	TypeSetup      ChunkType = "SETUP"
	TypeArch       ChunkType = "ARCHITECTURE"
	TypeFeature    ChunkType = "FEATURE"
	TypeIncomplete ChunkType = "INCOMPLETE"
	TypeDead       ChunkType = "DEAD_TEXT"
)

type Chunk struct {
	ID          string
	Type        ChunkType
	Title       string
	Body        string
	ActionItems []string
}

// Logic to parse README into semantic chunks
func ParseREADME(r io.Reader) ([]Chunk, error) {
	// (Implementation from Source [11-15])
	return []Chunk{}, nil // Stub for initialization
}
