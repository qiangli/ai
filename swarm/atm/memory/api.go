package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"atm/memory/internal/index"
)

type SearchResult struct {
	Content      string  `json:"content"`
	Path         string  `json:"path"`
	StartLine    int     `json:"start_line"`
	EndLine      int     `json:"end_line"`
	Score        float64 `json:"score"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	FallbackUsed bool    `json:"fallback_used"`
}

func Search(query string, cfg MemoryConfig) ([]SearchResult, error) {
	cfg.Defaults()
	if !cfg.Enabled {
		return []SearchResult{}, nil
	}
	mi, err := index.NewMemoryIndex(cfg)
	if err != nil {
		return nil, fmt.Errorf("new memory index: %w", err)
	}
	defer mi.Close()
	return mi.Search(query)
}

func Get(path string, fromLine int, linesCount int, cfg MemoryConfig) (string, error) {
	if fromLine < 1 {
		fromLine = 1
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	lineSlice := strings.Split(string(data), "\n")
	startIdx := fromLine - 1
	endIdx := startIdx + linesCount
	if endIdx > len(lineSlice) {
		endIdx = len(lineSlice)
	}
	if startIdx >= len(lineSlice) {
		return "", fmt.Errorf("line %d beyond EOF", fromLine)
	}
	content := strings.Join(lineSlice[startIdx:endIdx], "\n")
	return content, nil
}

func IndexWorkspace(rootPath string, cfg MemoryConfig) error {
	cfg.Defaults()
	mi, err := index.NewMemoryIndex(cfg)
	if err != nil {
		return fmt.Errorf("new memory index: %w", err)
	}
	defer mi.Close()
	return mi.IndexWorkspace(rootPath)
}