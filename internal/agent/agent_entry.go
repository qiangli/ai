package agent

import (
	"fmt"

	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/swarm"
)

var entrypointMap = map[string]swarm.Entrypoint{}

// PR entrypoint that generates and updates the instruction/system role prompt
func prPromptEntrypoint(vars *swarm.Vars, agent *swarm.Agent, input *swarm.UserInput) error {
	sub := agent.Name
	tplName := fmt.Sprintf("pr_%s_system_role", sub)
	schema := fmt.Sprintf("pr_%s_schema", sub)
	example := fmt.Sprintf("pr_%s_example", sub)

	data := map[string]any{
		"schema":         resource.Prompts[schema],
		"example":        resource.Prompts[example],
		"maxFiles":       8,
		"maxSuggestions": 8,
	}
	tpl, ok := resource.Prompts[tplName]
	if !ok {
		return fmt.Errorf("no such prompt resource: %s", tplName)
	}
	content, err := applyTemplate(tpl, data)
	if err != nil {
		return err
	}

	// update the system role prompt
	agent.Instruction = content
	return nil
}

func init() {
	entrypointMap["pr_system_role_prompt"] = prPromptEntrypoint
}
