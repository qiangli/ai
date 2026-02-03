package swarm

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/resource"
)

type Swarm struct {
	//
	vars *api.Vars
}

// set "workspace", "user", "input" in global env.
func New(vars *api.Vars) (*Swarm, error) {
	if vars.SessionID == "" {
		return nil, fmt.Errorf("Missing required session ID")
	}

	if vars.Base == "" {
		return nil, fmt.Errorf("app base required")
	}
	// required
	if vars.Workspace == nil {
		return nil, fmt.Errorf("app workspace not initialized")
	}
	if vars.User == nil {
		return nil, fmt.Errorf("user not authenticated")
	}
	if vars.Secrets == nil {
		return nil, fmt.Errorf("secret store not initialized")
	}
	if vars.OS == nil {
		return nil, fmt.Errorf("execution env not avalable")
	}

	//
	vars.Global = api.NewEnvironment()

	// export from os
	for _, v := range os.Environ() {
		kv := strings.SplitN(v, "=", 2)
		if len(kv) == 2 {
			vars.Global.Set(kv[0], kv[1])
		}
	}
	// preset
	vars.Global.Set("workspace", vars.Roots.Workspace.Path)
	vars.Global.Set("user", vars.User)
	vars.Global.Set("input", vars.Input)

	rootData := []byte("data:," + string(resource.RootAgentData))
	root, err := CreateAgent(context.TODO(), vars, nil, api.Packname("root"), rootData)
	if err != nil {
		return nil, err
	}
	vars.RootAgent = root

	sw := &Swarm{
		vars: vars,
	}

	return sw, nil
}

// set original "input" in global env.
func (sw *Swarm) Parse(ctx context.Context, input any) (api.ArgMap, error) {
	log.GetLogger(ctx).Debugf("argm: %+v\n", input)
	// save user raw input in env.
	// sw.vars.Global.Set("input", input)
	parsev := func(argv []string) (api.ArgMap, error) {
		if conf.IsAction(argv[0]) {
			cfg, err := GetInput(ctx, argv)
			if err != nil {
				return nil, err
			}
			// remove special trailing chars
			if cfg.Message != "" {
				argv = append(cfg.Args, "--stdin", cfg.Message)
			}
		}
		return conf.Parse(argv)
	}

	var argm api.ArgMap
	var err error
	switch input := input.(type) {
	case string:
		argv := conf.Argv(input)
		argm, err = parsev(argv)
	case []string:
		argm, err = parsev(input)
	default:
		argm, err = conf.Parse(input)
	}

	// default from user preference. update only if not set
	for k, v := range sw.vars.User.Settings {
		if _, ok := argm[k]; !ok {
			argm[k] = v
		}
	}
	return argm, err
}

func (sw *Swarm) Format(ctx context.Context, argm map[string]any) (*api.Result, error) {
	format, _ := api.GetStrProp("format", argm)
	if format == "" {
		format = "markdown"
	}
	var v string
	var tpl = resource.FormatFile(format)
	atm.CheckApplyTemplate(sw.vars.RootAgent.Template, tpl, argm)
	return &api.Result{
		Value: v,
	}, nil
}

func (sw *Swarm) Exec(ctx context.Context, input any) (*api.Result, error) {
	return sw.exec(ctx, sw.vars.RootAgent, input)
}

func (sw *Swarm) exec(ctx context.Context, parent *api.Agent, input any) (*api.Result, error) {
	argm, err := conf.Parse(input)
	if err != nil {
		return nil, err
	}
	return sw.execm(ctx, parent, argm)
}

// default action runner
func (sw *Swarm) execm(ctx context.Context, parent *api.Agent, argm map[string]any) (*api.Result, error) {
	log.GetLogger(ctx).Debugf("argm: %+v\n", argm)
	if msg, ok := argm["message"]; ok {
		sw.vars.Global.Set("message", msg)
	}

	return api.Exec(ctx, parent.Runner, argm)
}
