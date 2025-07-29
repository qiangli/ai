package swarm

import (
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestLoadAgentsConfig(t *testing.T) {
	app := &api.AppConfig{
		Base: "../internal/data/",
	}
	cfg, err := LoadAgentsConfig(app)
	if err != nil {
		t.Fatalf("Failed to load agent config: %v", err)
	}

	for _, v := range cfg {
		for _, agent := range v.Agents {
			if agent.Name == "" {
				t.Fatal("Agent name is empty")
			}
			if agent.Description == "" {
				t.Fatal("Agent description is empty")
			}
		}
	}
}
