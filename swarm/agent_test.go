package swarm

import (
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestLoadDefaultAgentConfig(t *testing.T) {
	app := &api.AppConfig{}
	cfg, err := LoadDefaultAgentConfig(app)
	if err != nil {
		t.Errorf("Failed to load default agent config: %v", err)
	}

	t.Logf("Successfully loaded default agent config:\n%+v\n", cfg)
}
