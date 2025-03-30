package agent

import (
	"fmt"
	"os"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/db"
	"github.com/qiangli/ai/internal/docker/gptr"
	"github.com/qiangli/ai/swarm/agent/resource"
	"github.com/qiangli/ai/swarm/api"
)

var entrypointMap = map[string]api.Entrypoint{}

func init() {
	entrypointMap["pr_system_role_prompt"] = prPromptEntrypoint
	entrypointMap["gptr_system_role_prompt"] = gptrPromptEntrypoint
	entrypointMap["sql_entry"] = sqlPromptEntrypoint
	entrypointMap["doc_compose_entry"] = docComposeEntrypoint
}

// PR entrypoint that generates and updates the instruction/system role prompt
func prPromptEntrypoint(vars *api.Vars, agent *api.Agent, input *api.UserInput) error {
	sub := baseCommand(agent.Name)
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

func gptrPromptEntrypoint(vars *api.Vars, agent *api.Agent, input *api.UserInput) error {
	data := map[string]any{
		"ReportTypes": gptr.ReportTypes,
		"Tones":       gptr.Tones,
	}
	vars.Extra["Data"] = data
	return nil
}

func sqlPromptEntrypoint(vars *api.Vars, agent *api.Agent, input *api.UserInput) error {
	data, err := db.GetDBInfo(vars.DBCred)
	if err != nil {
		return err
	}
	vars.Extra["SQL"] = data
	return nil
}

func docComposeEntrypoint(vars *api.Vars, agent *api.Agent, input *api.UserInput) error {
	// read the template
	var temp []byte
	if input.Template == "" {
		return internal.NewUserInputError("no template file provided")
	}
	var err error
	temp, err = os.ReadFile(input.Template)
	if err != nil {
		return err
	}
	if len(temp) == 0 {
		return internal.NewUserInputError("empty template file")
	}

	// read the draft
	draft, err := input.FileContent()
	if err != nil {
		return err
	}
	if len(draft) == 0 {
		return internal.NewUserInputError("empty draft content")
	}

	data := map[string]string{
		"Template": string(temp),
		"Draft":    draft,
	}
	vars.Extra["Doc"] = data
	return nil
}
