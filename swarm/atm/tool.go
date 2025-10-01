package atm

import (
	"context"
	"fmt"
	"strings"

	// log "github.com/sirupsen/logrus"

	// "github.com/qiangli/ai/swarm/agent/api/entity"
	// "github.com/qiangli/ai/swarm/agent/internal/db"
	// "github.com/qiangli/ai/swarm/agent/internal/hub/conf"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

type toolCaller struct {
	agent *api.Agent
	vars  *api.Vars
	user  *api.User

	//
}

var secrets api.SecretStore

func NewToolCaller(auth *api.User, owner string) api.ToolCaller {

	funcKit := &FuncKit{
		User: auth,
	}

	faasKit := &FaasKit{
		// cfg: cfg,
	}

	toResult := func(v any) *api.Result {
		if r, ok := v.(*api.Result); ok {
			return r
		}
		if s, ok := v.(string); ok {
			return &api.Result{
				Value: s,
			}
		}
		return &api.Result{
			Value: fmt.Sprintf("%v", v),
		}
	}

	dispatch := func(ctx context.Context, vars *api.Vars, v *api.ToolFunc, args map[string]any) (*api.Result, error) {
		switch v.Type {
		case api.ToolTypeMcp:
			mcpKit := &McpKit{
				token: func() (string, error) {
					return secrets.Get(owner, v.ApiKey)
					// return db.LoadApiKey(owner, v.ApiKey, cfg.ApiMasterKey)
				},
			}
			out, err := mcpKit.callTool(ctx, vars, v, args)
			return &api.Result{
				Value: out,
			}, err
		case api.ToolTypeSystem:
			return nil, fmt.Errorf("local system tool not supported: %s", v.ID())
		case api.ToolTypeWeb:
			webKit := &WebKit{
				apiKey: func() (string, error) {
					return secrets.Get(owner, v.ApiKey)
					// return db.LoadApiKey(owner, v.ApiKey, cfg.ApiMasterKey)
				},
			}
			out, err := webKit.callTool(ctx, vars, v, args)
			if err != nil {
				return &api.Result{}, fmt.Errorf("failed to call web tool %s %s: %w", v.Kit, v.Name, err)
			}
			return toResult(out), nil
		case api.ToolTypeFaas:
			return faasKit.callTool(ctx, vars, v, args)
		case api.ToolTypeFunc:
			out, err := funcKit.callTool(ctx, vars, v, args)
			if err != nil {
				return &api.Result{}, fmt.Errorf("failed to call function tool %s %s: %w", v.Kit, v.Name, err)
			}
			return toResult(out), nil
		}

		return nil, fmt.Errorf("no such tool: %s", v.ID())
	}

	return func(vars *api.Vars, agent *api.Agent) func(context.Context, string, map[string]any) (*api.Result, error) {
		toolMap := make(map[string]*api.ToolFunc)
		for _, v := range agent.Tools {
			toolMap[v.ID()] = v
		}

		return func(ctx context.Context, tid string, args map[string]any) (*api.Result, error) {
			// swarmlog.GetLogger(ctx).Debugf("run tool: %s %+v\n", tid, args)
			v, ok := toolMap[tid]
			if !ok {
				return nil, fmt.Errorf("tool not found: %s", tid)
			}

			log.GetLogger(ctx).Infof("⣿ %s:%s %+v\n", v.Kit, v.Name, args)

			result, err := dispatch(ctx, vars, v, args)

			if err != nil {
				log.GetLogger(ctx).Errorf("✗ error: %v\n", err)
			} else {
				log.GetLogger(ctx).Infof("✔ %s \n", head(result.String(), 180))
			}

			return result, err
		}
	}
}

// head trims the string to the maxLen and replaces newlines with /.
func head(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "/")
	s = strings.Join(strings.Fields(s), " ")
	s = strings.TrimSpace(s)
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}
