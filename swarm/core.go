package swarm

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/resource"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

type Swarm struct {
	// session id
	ID string

	User *api.User

	Secrets api.SecretStore

	Assets api.AssetManager

	Tools api.ToolSystem

	Adapters api.AdapterRegistry

	Blobs api.BlobStore

	OS        vos.System
	Workspace vfs.Workspace
	History   api.MemStore
	Log       api.CallLogger

	//
	Vars *api.Vars
}

func (sw *Swarm) Init(rte *api.ActionRTEnv) error {
	if rte == nil {
		return fmt.Errorf("Action RT env required")
	}
	if rte.Base == "" {
		return fmt.Errorf("app base required")
	}
	// required
	if rte.Workspace == nil {
		return fmt.Errorf("app workspace not initialized")
	}
	if rte.User == nil {
		return fmt.Errorf("user not authenticated")
	}
	if rte.Secrets == nil {
		return fmt.Errorf("secret store not initialized")
	}
	if rte.OS == nil {
		return fmt.Errorf("execution env not avalable")
	}

	sw.Vars = api.NewVars()
	sw.Vars.RTE = rte

	// preset
	sw.Vars.Global.Set("workspace", sw.Vars.RTE.Roots.Workspace)
	sw.Vars.Global.Set("user", sw.Vars.RTE.User)

	rootData := []byte("data:," + string(resource.RootAgentData))
	root, err := sw.CreateAgent(context.TODO(), nil, api.Packname("root"), rootData)
	if err != nil {
		return err
	}
	sw.Vars.RootAgent = root

	return nil
}

func (sw *Swarm) CreateAgent(ctx context.Context, parent *api.Agent, packname api.Packname, config []byte) (*api.Agent, error) {
	var loader = NewConfigLoader(sw.Vars.RTE)

	if config != nil {
		// load data
		if err := loader.LoadContent(string(config)); err != nil {
			return nil, err
		}
	}

	agent, err := loader.Create(ctx, packname)
	if err != nil {
		return nil, err
	}

	// init setup

	// agent.Parent = parent
	// agent.Runner = NewAgentToolRunner(sw, sw.User.Email, agent)
	// agent.Shell = NewAgentScriptRunner(sw, agent)
	// agent.Template = NewTemplate(sw, agent)

	// TODO optimize
	// embeded
	add := func(p, a *api.Agent) {
		a.Parent = p
		a.Runner = NewAgentToolRunner(sw, sw.User.Email, a)
		a.Shell = NewAgentScriptRunner(sw, a)
		a.Template = NewTemplate(sw, a)
	}

	var addAll func(*api.Agent, *api.Agent)
	addAll = func(p, a *api.Agent) {
		for _, v := range a.Embed {
			addAll(p, v)
		}
		add(p, a)
	}

	addAll(parent, agent)

	return agent, nil
}

// copy values from src to dst after applying templates if requested
// skip unless override is true
// var in src template can reference global env
func (sw *Swarm) mapAssign(_ context.Context, agent *api.Agent, dst, src map[string]any, override bool) error {
	if len(src) == 0 {
		return nil
	}
	var data = make(map[string]any)
	maps.Copy(data, sw.Vars.Global.GetAllEnvs())
	for key, val := range src {
		if _, ok := dst[key]; ok && !override {
			continue
		}
		// go template value support
		if atm.IsTemplate(val) {
			maps.Copy(data, dst)
			if resolved, err := atm.CheckApplyTemplate(agent.Template, val.(string), data); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		dst[key] = val
	}
	return nil
}

func (sw *Swarm) Parse(ctx context.Context, input any) (api.ArgMap, error) {
	// parse special chars: - }}
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

	switch input := input.(type) {
	case string:
		argv := conf.Argv(input)
		return parsev(argv)
	case []string:
		return parsev(input)
	}
	return conf.Parse(input)
}

func (sw *Swarm) Format(ctx context.Context, argm map[string]any) (*api.Result, error) {
	format, _ := api.GetStrProp("format", argm)
	if format == "" {
		format = "markdown"
	}
	var v string
	var tpl = resource.FormatFile(format)
	atm.CheckApplyTemplate(sw.Vars.RootAgent.Template, tpl, argm)
	return &api.Result{
		Value: v,
	}, nil
}

func (sw *Swarm) Exec(ctx context.Context, input any) (*api.Result, error) {
	return sw.exec(ctx, sw.Vars.RootAgent, input)
}

func (sw *Swarm) exec(ctx context.Context, parent *api.Agent, input any) (*api.Result, error) {
	argm, err := conf.Parse(input)
	if err != nil {
		return nil, err
	}
	return sw.execm(ctx, parent, argm)
}

// default action runner
func (sw *Swarm) execm(ctx context.Context, agent *api.Agent, argm map[string]any) (*api.Result, error) {
	log.GetLogger(ctx).Debugf("argm: %+v\n", argm)

	am := api.ArgMap(argm)
	id := am.Kitname().ID()
	if id == "" {
		// required
		// kit is optional for system command
		return nil, fmt.Errorf("missing action id: %+v", argm)
	}
	v, err := agent.Runner.Run(ctx, id, argm)
	if err != nil {
		return nil, err
	}
	result := api.ToResult(v)
	return result, nil
}

// inherit parent tools including embedded agents
// TODO cache
func (sw *Swarm) buildAgentToolMap(agent *api.Agent) map[string]*api.ToolFunc {
	toolMap := make(map[string]*api.ToolFunc)
	if agent == nil {
		return toolMap
	}
	// inherit tools of embedded agents
	for _, agent := range agent.Embed {
		for _, v := range agent.Tools {
			toolMap[v.ID()] = v
		}
	}
	// the active agent
	for _, v := range agent.Tools {
		toolMap[v.ID()] = v
	}
	return toolMap
}

func (sw *Swarm) callTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {

	log.GetLogger(ctx).Infof("⣿ %s:%s %+v\n", tf.Kit, tf.Name, formatArgs(args))

	var entry = api.CallLogEntry{
		Kit:       tf.Kit,
		Name:      tf.Name,
		Arguments: args,
		Started:   time.Now(),
	}
	if agent != nil {
		entry.Agent = string(api.NewPackname(agent.Pack, agent.Name))
	}

	result, err := sw.dispatch(ctx, agent, tf, args)

	entry.Ended = time.Now()

	if err != nil {
		entry.Error = err
		log.GetLogger(ctx).Errorf("✗ error: %v\n", err)
	} else {
		// in case nil is returned by the tools
		if result == nil {
			result = &api.Result{}
		}
		entry.Result = result
		log.GetLogger(ctx).Infof("✔ %s (%s)\n", tf.ID(), head(result.String(), 180))
		log.GetLogger(ctx).Debugf("details:\n%s\n", result.String())
	}

	sw.Log.Save(&entry)

	return result, err
}

func (sw *Swarm) dispatch(ctx context.Context, agent *api.Agent, v *api.ToolFunc, args api.ArgMap) (*api.Result, error) {
	// command
	if v.Type == api.ToolTypeBin {
		out, err := atm.ExecCommand(ctx, sw.OS, sw.Vars, v.Name, nil)
		if err != nil {
			return nil, err
		}
		return &api.Result{
			Value: out,
		}, nil
	}

	// ai
	if v.Type == api.ToolTypeAI {
		aiKit := NewAIKit(sw, agent)
		out, err := aiKit.Call(ctx, sw.Vars, v, args)
		if err != nil {
			return nil, err
		}
		return api.ToResult(out), nil
	}

	// agent tool
	if v.Type == api.ToolTypeAgent {
		aiKit := NewAIKit(sw, agent)
		var in map[string]any
		if len(v.Arguments) > 0 {
			in = make(map[string]any)
			maps.Copy(in, v.Arguments)
			maps.Copy(in, args)
		} else {
			in = args
		}
		in["agent"] = v.Name
		return aiKit.SpawnAgent(ctx, sw.Vars, "", in)
	}

	// tools
	kit, err := sw.Tools.GetKit(v)
	if err != nil {
		return nil, err
	}

	env := &api.ToolEnv{
		Agent: agent,
	}
	out, err := kit.Call(ctx, sw.Vars, env, v, args)

	if err != nil {
		return nil, err
	}
	return api.ToResult(out), nil
}

func (sw *Swarm) LoadScript(v string) (string, error) {
	return api.LoadURIContent(sw.Workspace, v)
}
