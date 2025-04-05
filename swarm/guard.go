package swarm

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
)

//go:embed resource/shell_security_system.md
var shellSecuritySystemRole string

//go:embed resource/shell_security_user.md
var shellSecurityUserRole string

const permissionDenied = "Permission denied."

type CommandCheck struct {
	Command string `json:"command"`
	Safe    bool   `json:"safe"`
}

// evaluateCommand consults LLM to evaluate the safety of a command
func evaluateCommand(ctx context.Context, vars *api.Vars, command string, args []string) (bool, error) {
	if vars.Config.Unsafe {
		log.Infof("⚠️ unsafe mode - skipping security check\n")
		return true, nil
	}

	log.Infof("🔒 checking %s %+v\n", command, args)

	instruction, err := applyTemplate(shellSecuritySystemRole, vars, nil)
	if err != nil {
		return false, err
	}

	vars.Extra["Command"] = command
	vars.Extra["Args"] = strings.Join(args, " ")
	query, err := applyTemplate(shellSecurityUserRole, vars, nil)
	if err != nil {
		return false, err
	}

	runTool := func(ctx context.Context, name string, args map[string]any) (*api.Result, error) {
		log.Debugf("run tool: %s %+v\n", name, args)
		out, err := CallTool(ctx, vars, name, args)
		return out, err
	}

	// TODO default model
	model, ok := vars.Models[api.L1]
	if !ok {
		model = vars.Models[api.L2]
	}
	if model == nil {
		return false, fmt.Errorf("no model found L1/L2")
	}

	req := &api.LLMRequest{
		ModelType: model.Type,
		Model:     model.Name,
		BaseUrl:   model.BaseUrl,
		ApiKey:    model.ApiKey,
		Messages: []*api.Message{
			{
				Role:    api.RoleSystem,
				Content: instruction,
			},
			{
				Role:    api.RoleUser,
				Content: query,
			},
		},
		Tools:    systemTools,
		RunTool:  runTool,
		MaxTurns: vars.Config.MaxTurns,
	}

	log.Debugf("evaluateCommand:\n%s %v\n", command, args)

	resp, err := llm.Send(ctx, req)
	if err != nil {
		return false, err
	}

	var check CommandCheck
	if err := json.Unmarshal([]byte(resp.Content), &check); err != nil {
		return false, fmt.Errorf("%s %s: %s, %s", command, strings.Join(args, " "), permissionDenied, resp.Content)
	}

	if check.Safe {
		log.Infof("✅ safe\n")
	} else {
		log.Errorf("❌ unsafe\n")
	}

	log.Debugf("evaluateCommand:\n%+v\n", check)

	return check.Safe, nil
}
