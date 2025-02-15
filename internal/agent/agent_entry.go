package agent

import (
	"fmt"

	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/docker/gptr"
	"github.com/qiangli/ai/internal/swarm"
)

var entrypointMap = map[string]swarm.Entrypoint{}

// PR entrypoint that generates and updates the instruction/system role prompt
func prPromptEntrypoint(vars *swarm.Vars, agent *swarm.Agent, input *swarm.UserInput) error {
	sub := agent.Name
	schema := fmt.Sprintf("pr_%s_schema", sub)
	example := fmt.Sprintf("pr_%s_example", sub)

	data := map[string]any{
		"Schema":         resource.Prompts[schema],
		"Example":        resource.Prompts[example],
		"MaxFiles":       8,
		"MaxSuggestions": 8,
	}
	vars.Extra["PR"] = data
	return nil
}

func gptrPromptEntrypoint(vars *swarm.Vars, agent *swarm.Agent, input *swarm.UserInput) error {
	data := map[string]any{
		"ReportTypes": gptr.ReportTypes,
		"Tones":       gptr.Tones,
	}
	vars.Extra["Data"] = data
	return nil
}

func init() {
	entrypointMap["pr_system_role_prompt"] = prPromptEntrypoint
	entrypointMap["gptr_system_role_prompt"] = gptrPromptEntrypoint
}
