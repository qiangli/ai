package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

type AgentToolRunner struct {
	user    string
	vars    *api.Vars
	agent   *api.Agent
	toolMap map[string]*api.ToolFunc
}

func NewAgentToolRunner(vars *api.Vars, agent *api.Agent) api.ActionRunner {
	toolMap := buildAgentToolMap(agent)
	return &AgentToolRunner{
		vars:    vars,
		user:    vars.User.Email,
		agent:   agent,
		toolMap: toolMap,
	}
}

func (r *AgentToolRunner) loadYaml(tid string, base string, script string) (*api.ToolFunc, error) {
	kit, name := api.Kitname(tid).Decode()
	// /agent:
	if kit == string(api.ToolTypeAgent) {
		ac, err := conf.LoadAgentsData([][]byte{[]byte(script)})
		if err != nil {
			return nil, err
		}
		pack, sub := api.Packname(name).Decode()
		if ac.Pack != pack {
			return nil, fmt.Errorf("Invalid pack name: %s. config: %s", pack, ac.Pack)
		}
		ac.Store = r.vars.Workspace
		ac.BaseDir = base
		for _, v := range ac.Agents {
			if (sub == "" && v.Name == pack) || sub == v.Name {
				v, err := conf.LoadAgentTool(ac, pack, v.Name)
				if err == nil && len(v) == 1 {
					v[0].Config = ac
					return v[0], nil
				}
				return nil, fmt.Errorf("Error loading %s/%s", pack, sub)
			}
		}
	} else {
		// /kit:tool
		tc, err := conf.LoadToolData([][]byte{[]byte(script)})
		if err != nil {
			return nil, err
		}
		tc.Store = r.vars.Workspace
		tc.BaseDir = base
		tools, err := conf.LoadTools(tc, r.user, r.vars.Secrets)
		if err != nil {
			return nil, err
		}
		for _, v := range tools {
			if v.Kit == kit && v.Name == name {
				v.Config = tc
				return v, nil
			}
			// default
			if v.Kit == kit && name == "" && v.Name == kit {
				v.Config = tc
				return v, nil
			}
		}
	}
	return nil, nil
}

func (r *AgentToolRunner) loadScript(uri string) (string, error) {
	return api.LoadURIContent(r.vars.Workspace, uri)
	// if strings.HasPrefix(uri, "data:") {
	// 	return api.DecodeDataURL(uri)
	// } else {
	// 	var f = uri
	// 	if strings.HasPrefix(f, "file:") {
	// 		v, err := url.Parse(f)
	// 		if err != nil {
	// 			return "", err
	// 		}
	// 		f = v.Path
	// 	}
	// 	data, err := r.vars.Workspace.ReadFile(f, nil)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	return string(data), nil
	// }
}

// try load from provided script file first,
// if not found, continue to load from other sources
func (r *AgentToolRunner) loadTool(tid string, args map[string]any) (*api.ToolFunc, error) {
	// load tool from content
	if s, ok := args["script"]; ok {
		s := api.ToString(s)
		ext := path.Ext(s)
		switch ext {
		case ".sh", ".bash":
			// continue
		case ".yaml", ".yml":
			cfg, err := r.loadScript(s)
			if err != nil {
				return nil, err
			}
			base := filepath.Dir(s)
			if strings.HasPrefix(base, "file:") {
				v, err := url.Parse(base)
				if err != nil {
					return nil, err
				}
				base = v.Path
			}
			tf, err := r.loadYaml(tid, base, cfg)
			if err != nil {
				return nil, err
			}
			if tf != nil {
				return tf, nil
			}
			// return nil, fmt.Errorf("%q not found in script: %s", tid, s)
			// continue to load tid
		case ".txt", ".md", ".markdown":
			// feed text file as query content
			cfg, err := r.loadScript(s)
			if err != nil {
				return nil, err
			}
			old := args["content"]
			args["content"] = api.Cat(api.ToString(old), Tail(cfg, 1), "\n###\n")
			// continue to load tid
		case ".json", ".jsonc":
			// decode json as additional arguments
			cfg, err := r.loadScript(s)
			if err != nil {
				return nil, err
			}
			obj := Tail(cfg, 1)
			obj = strings.TrimSpace(obj)
			if err := json.Unmarshal([]byte(obj), &args); err != nil {
				return nil, err
			}
			// continue to load tid
		default:
			// ignore script property
		}
	}

	// inline
	v, ok := r.toolMap[tid]
	if ok {
		return v, nil
	}

	// load external
	tools, err := conf.LoadToolFunc(r.user, tid, r.vars.Secrets, r.vars.Assets)
	if err != nil {
		return nil, err
	}
	for _, v := range tools {
		id := v.ID()
		if id == tid {
			return v, nil
		}
	}
	return nil, fmt.Errorf("invalid tool: %s", tid)
}

// Lookup aciton for tid or kit:name in the args if tid is not provided
func (r *AgentToolRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
	var kit, name string
	if tid == "" {
		kit = api.ToString(args["kit"])
		name = api.ToString(args["name"])
		if kit == "agent" {
			pack := api.ToString(args["pack"])
			if pack == "" {
				return nil, fmt.Errorf("missing pack for agent tool: %s", name)
			}
			name = pack + "/" + name
		}
		tid = api.NewKitname(kit, name).ID()
	} else {
		kit, name = api.Kitname(tid).Decode()
	}

	// /bin/command (local system)
	// sh:*
	// agent:*
	// kit:*

	// /alias:name --option name="command line"
	// name to lookup the command in args
	// TODO alias referencing alias
	// alias use the same args in the same scope
	if kit == string(api.ToolTypeAlias) {
		if name == "" {
			return nil, fmt.Errorf("Invalid alias action. missing name. /alias:NAME --option NAME='action to run'")
		}

		// TODO expand to general action not just for alias
		// execute if action runner is provided
		if v, found := args[name]; found {
			if runner, ok := v.(api.ActionRunner); ok {
				return runner.Run(ctx, tid, args)
			}
		}

		//
		alias, err := api.GetStrProp(name, args)
		if err != nil {
			return nil, fmt.Errorf("Failed to resolve alias: %s, error: %v", name, err)
		}
		if alias == "" {
			return nil, fmt.Errorf("Alias not found: %s", name)
		}

		argv := conf.Argv(alias)
		if len(argv) == 0 {
			return nil, fmt.Errorf("Invalid alias %q: %s", name, alias)
		}
		//
		if conf.IsAction(argv[0]) {
			nargs, err := conf.Parse(argv)
			if err != nil {
				return nil, err
			}
			maps.Copy(args, nargs)
			kit, name = api.Kitname(argv[0]).Decode()
		} else {
			// exec alias as system command line
			kit = string(api.ToolTypeBin)
			args["command"] = alias
		}
	}

	// this ensures kit:name is in internal kit__name format
	tid = api.NewKitname(kit, name).ID()

	var tf *api.ToolFunc

	// system command
	// /bin/*
	// /bin: pseudo kit name "bin"
	// support direct execution of system command using the standad syntax
	// e.g. /bin/ls without the trigger word "ai" and slash command toolkit "/kit:name" syntax
	// i.e. equivalent to: /sh:exec --arg command="ls -al /tmp"
	if kit == "" || kit == string(api.ToolTypeBin) {
		// system command action
		cmd, _ := api.GetStrProp("command", args)
		if cmd == "" {
			return nil, fmt.Errorf("no kit specified. system command is required")
		}
		tf = &api.ToolFunc{
			Type: api.ToolTypeBin,
			Kit:  string(api.ToolTypeBin),
			Name: cmd,
		}
	} else {
		// agent/tool action
		v, err := r.loadTool(tid, args)
		if err != nil {
			return nil, err
		}
		tf = v
	}

	// handle 'output'
	// consume the first time it is seen.
	// similar to command line redirect ">"
	output, _ := api.GetStrProp("output", args)
	if output != "" {
		tf.Output = output
		delete(args, "output")
	}

	// run the action
	uctx := context.WithValue(ctx, api.SwarmUserContextKey, r.user)
	result, err := r.callTool(uctx, tf, args)

	if err != nil {
		return "", err
	}
	if result == nil {
		result = &api.Result{}
	}
	return result, err
}

func (r *AgentToolRunner) callTool(ctx context.Context, tf *api.ToolFunc, input map[string]any) (*api.Result, error) {
	var args map[string]any
	if len(tf.Arguments) > 0 {
		args = make(map[string]any)
		maps.Copy(args, tf.Arguments)
		maps.Copy(args, input)
	} else {
		args = input
	}
	log.GetLogger(ctx).Infof("⣿ %s:%s %+v\n", tf.Kit, tf.Name, api.FormatArgMap(args))

	const outformat = `
	The output is too large and has been saved to %q.
	Use the 'fs:read_file' tool to access it with path, offset, and limit parameters.
	Total size: %v
	`

	// save oversized output
	saveOutput := func(val string) (string, error) {
		size, _ := api.GetIntProp("max_output_size", args)
		if size <= 0 {
			size = 512000
		}
		if len(val) < size {
			return val, nil
		}
		tid := api.NewPackname(tf.Kit, tf.Name).ID()
		rid := uuid.NewString()
		dir := filepath.Join(r.vars.Roots.Workspace.Path, "oversize", tid)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", nil
		}
		outfile := filepath.Join(dir, rid+"-output.txt")
		if err := os.WriteFile(outfile, []byte(val), 0600); err != nil {
			return "", err
		}
		return fmt.Sprintf(outformat, outfile, size), nil
	}
	var entry = api.CallLogEntry{
		Agent:     string(api.NewPackname(r.agent.Pack, r.agent.Name)),
		Kit:       tf.Kit,
		Name:      tf.Name,
		Arguments: args,
		Started:   time.Now(),
	}

	result, err := r.dispatch(ctx, tf, args)

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

		if result.State == api.StateTransfer {
			if result.NextAgent == "" {
				return nil, fmt.Errorf("Taget agent is required for transfer")
			}
			log.GetLogger(ctx).Infof("➡️ %s:%s @%s\n", tf.Kit, tf.Name, result.NextAgent)
		} else {
			if tf.Output != "" {
				out := r.output(tf.Output, result.Value)
				result.Value = out
			}
			val, err := saveOutput(result.Value)
			result.Value = val
			entry.Error = err
			log.GetLogger(ctx).Infof("✔ %s:%s (%s %v)\n", tf.Kit, tf.Name, Head(val, 180), len(val))
		}

		log.GetLogger(ctx).Debugf("details:\n%s\n", result.Value)
	}

	r.vars.Log.Save(&entry)

	return result, err
}

func (r *AgentToolRunner) dispatch(ctx context.Context, tf *api.ToolFunc, args api.ArgMap) (*api.Result, error) {
	// command
	if tf.Type == api.ToolTypeBin {
		out, err := atm.ExecCommand(ctx, r.vars.OS, r.vars, tf.Name, nil)
		if err != nil {
			return nil, err
		}
		return &api.Result{
			Value: out,
		}, nil
	}

	// ai
	if tf.Type == api.ToolTypeAI {
		aiKit := NewAIKit(r.vars)
		out, err := aiKit.Call(ctx, r.vars, r.agent, tf, args)
		if err != nil {
			return nil, err
		}
		return api.ToResult(out), nil
	}

	// agent tool
	if tf.Type == api.ToolTypeAgent {
		aiKit := NewAIKit(r.vars)
		// var in map[string]any
		// if len(v.Arguments) > 0 {
		// 	in = make(map[string]any)
		// 	maps.Copy(in, v.Arguments)
		// 	maps.Copy(in, args)
		// } else {
		// 	in = args
		// }
		args["agent"] = tf.Name
		return aiKit.SpawnAgent(ctx, r.vars, r.agent, nil, args)
	}

	// tools
	kit, err := r.vars.Tools.GetKit(tf)
	if err != nil {
		return nil, err
	}

	out, err := kit.Call(ctx, r.vars, r.agent, tf, args)

	if err != nil {
		return nil, err
	}
	return api.ToResult(out), nil
}

// inherit parent tools including embedded agents
// TODO cache
func buildAgentToolMap(agent *api.Agent) map[string]*api.ToolFunc {
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

func (r *AgentToolRunner) output(output string, content string) string {
	switch output {
	case "none", "/dev/null":
		return ""
	case "console", "":
		return content
	default:
		// uri
		// [scheme:][//[userinfo@]host][/]path[?query][#fragment]
		uri, err := url.Parse(output)
		if err != nil {
			return fmt.Sprintf("Invalid output scheme: %v", output)
		}
		// env:key
		// scheme:opaque[?query][#fragment]
		if uri.Scheme == "env" {
			key := uri.Opaque
			key = strings.ReplaceAll(key, "/", "__")
			key = strings.ReplaceAll(key, ":", "__")
			key = strings.ReplaceAll(key, "-", "_")
			r.vars.Global.Set(key, content)
			r.vars.OS.Setenv(key, content)
			return fmt.Sprintf("env %q set", key)
		}
		//
		// file:///path
		// file:path
		if uri.Scheme == "file" {
			file := output[5:]
			resolved, err := api.ResolvePath(file)
			if err != nil {
				return fmt.Sprintf("Failed to resolve file path: %q. %v", file, err)
			}
			err = r.vars.Workspace.WriteFile(resolved[0], []byte(content))
			if err != nil {
				return fmt.Sprintf("Failed to save: %q. %v", file, err)
			}
			return fmt.Sprintf("File saved successfully: %s", file)
		}
	}
	return fmt.Sprintf("Output scheme not supported: %s", output)
}
