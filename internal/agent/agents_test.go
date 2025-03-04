package agent

import (
	"testing"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/swarm"
)

func TestCreateAgent(t *testing.T) {
	var cfg = &internal.AppConfig{}

	sw, err := swarm.NewSwarm(cfg)
	if err != nil {
		t.Fatal(err)
	}

	sw.AgentConfigMap = agentConfigMap
	sw.AgentToolMap = agentToolMap
	sw.ResourceMap = resourceMap
	sw.TemplateFuncMap = tplFuncMap
	sw.AdviceMap = adviceMap
	sw.EntrypointMap = entrypointMap

	var name = "script"
	var input = &swarm.UserInput{}

	agent, err := sw.Create(name, input)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Swarm config loaded successfully %+v", agent)
}
