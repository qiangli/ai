package memory

import "time"

type MemoryConfig struct {
	Enabled    bool     `json:"enabled"`
	Provider   string   `json:"provider"` // "openai", "gemini", "local"
	Model      string   `json:"model"`
	Fallback   string   `json:"fallback"`
	StorePath  string   `json:"store_path"`
	ExtraPaths []string `json:"extra_paths"`
	Query      struct {
		MaxResults int `json:"max_results"`
		Hybrid     struct {
			Enabled             bool    `json:"enabled"`
			VectorWeight        float64 `json:"vector_weight"`
			TextWeight          float64 `json:"text_weight"`
			CandidateMultiplier int     `json:"candidate_multiplier"`
		} `json:"hybrid"`
	} `json:"query"`
	Sync struct {
		Watch    []string `json:"watch"`
		Debounce string   `json:"debounce"`
	} `json:"sync"`
	Cache struct {
		Enabled bool   `json:"enabled"`
		TTL     string `json:"ttl"`
	} `json:"cache"`
}

func (c *MemoryConfig) Defaults() {
	if c.Query.MaxResults == 0 {
		c.Query.MaxResults = 5
	}
	c.Query.Hybrid.Enabled = true
	if c.Query.Hybrid.VectorWeight == 0 {
		c.Query.Hybrid.VectorWeight = 0.7
	}
	if c.Query.Hybrid.TextWeight == 0 {
		c.Query.Hybrid.TextWeight = 0.3
	}
	if c.Query.Hybrid.CandidateMultiplier == 0 {
		c.Query.Hybrid.CandidateMultiplier = 3
	}
	if c.Sync.Debounce == "" {
		c.Sync.Debounce = "1.5s"
	}
}
