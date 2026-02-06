package chunk

import (
	"math"
	"os"
	"path/filepath"
	"strings"
)

const (
	targetChars     = 1600
	overlapFraction = 0.2
	minChunkLines   = 5
)

type Chunk struct {
	Path      string
	StartLine int
	EndLine   int
	Content   string
}

func ChunksFromFile(path string) ([]Chunk, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(data), "\n")
	return ChunksFromLines(path, lines)
}

func ChunksFromLines(path string, lines []string) []Chunk {
	var chunks []Chunk
	start := 0
	for start < len(lines) {
		end := start
		accum := 0
		for end < len(lines) {
			lineLen := len(lines[end])
			if end > start {
				lineLen += 1 // \\n
			}
			if accum+lineLen > targetChars && end-start >= minChunkLines {
				break
			}
			accum += lineLen
			end++
		}
		if end == start {
			end++
		}
		content := strings.Join(lines[start:end], "\\n")
		overlap := int(float64(end-start) * overlapFraction)
		if overlap < 2 {
			overlap = 2
		}
		chunks = append(chunks, Chunk{
			Path:      path,
			StartLine: start + 1,
			EndLine:   end,
			Content:   content,
		})
		start = end - overlap
		if start < 0 {
			start = 0
		}
	}
	return chunks
}
