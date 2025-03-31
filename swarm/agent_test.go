package swarm

import (
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestLoadAgentsConfig(t *testing.T) {
	app := &api.AppConfig{}
	cfg, err := LoadAgentsConfig(app, resourceBase)
	if err != nil {
		t.Fatalf("Failed to load agent config from %s: %v", resourceBase, err)
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

func TestLoadDefaultAgentsConfig(t *testing.T) {
	app := &api.AppConfig{}
	cfg, err := LoadDefaultAgentsConfig(app)
	if err != nil {
		t.Errorf("Failed to load default agent config: %v", err)
	}
	for _, v := range cfg {
		for _, agent := range v.Agents {
			if agent.Name == "" {
				t.Fatal("Default Agent name is empty")
			}
			if agent.Description == "" {
				t.Fatal("Default Agent description is empty")
			}
		}
	}
}
