package conf

import (
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
)

// Load and resolve path for roots and assets
func Load(base string) (*api.DHNTConfig, error) {
	cfg, err := api.LoadDHNTConfig(filepath.Join(base, "dhnt.json"))
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
