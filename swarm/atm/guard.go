package atm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/qiangli/ai/internal/bubble"
	"github.com/qiangli/ai/internal/bubble/confirm"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/resource"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/vos"
)

type ContextKey string

const ModelsContextKey ContextKey = "eval-models"

const permissionDenied = "Permission denied."

type CommandCheck struct {
	Command string `json:"command"`
	Safe    bool   `json:"safe"`
}

// EvaluateCommand consults LLM to evaluate the safety of a command
func EvaluateCommand(ctx context.Context, vs vos.System, vars *api.Vars, command string, args []string) (bool, error) {
	if vars.Config.Unsafe {
		log.GetLogger(ctx).Infof("⚠️ unsafe mode - skipping security check\n")
		return true, nil
	}

	log.GetLogger(ctx).Infof("🔒 checking %s %+v\n", command, args)

	var data = make(map[string]any)
	data["OS"] = runtime.GOOS
	data["Shell"] = nvl(os.Getenv("SHELL"), "/bin/sh")
	data["Command"] = command
	data["Args"] = strings.Join(args, " ")

	instruction, err := applyTemplate(resource.ShellSecuritySystemRole, data, nil)
	if err != nil {
		return false, err
	}

	query, err := applyTemplate(resource.ShellSecurityUserRole, data, nil)
	if err != nil {
		return false, err
	}

	var model = ctx.Value(ModelsContextKey).(*api.Model)
	req := &llm.Request{
		Model: model,
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
		// Tools:    vars.Config.SystemTools,
		// RunTool:  runTool,
		MaxTurns: vars.Config.MaxTurns,
		Vars:     vars,
	}

	log.GetLogger(ctx).Debugf("evaluateCommand:\n%s %v\n", command, args)

	resp, err := adapter.Chat(ctx, req)
	if err != nil {
		return false, err
	}
	if resp.Result == nil {
		return false, fmt.Errorf("empty respone")
	}

	var check CommandCheck
	if err := json.Unmarshal([]byte(resp.Result.Value), &check); err != nil {
		return false, fmt.Errorf("%s %s: %s, %s", command, strings.Join(args, " "), permissionDenied, resp.Result.Value)
	}

	if check.Safe {
		log.GetLogger(ctx).Infof("✔ safe\n")
	} else {
		log.GetLogger(ctx).Errorf("❌ unsafe\n")
		log.GetLogger(ctx).Infof("%s %v\n", command, strings.Join(args, " "))
		if answer, err := bubble.Confirm("Continue?"); err == nil && answer == confirm.Yes {
			check.Safe = true
		}
	}

	log.GetLogger(ctx).Debugf("evaluateCommand:\n%+v\n", check)

	return check.Safe, nil
}
