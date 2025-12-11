package conf

import (
	"path/filepath"

	"github.com/qiangli/ai/swarm/api"
)

// default assets with resource.json and standard
func Load(base string) (*api.DHNTConfig, error) {
	cfg, err := api.LoadDHNTConfig(filepath.Join(base, "dhnt.json"))
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
