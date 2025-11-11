package swarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

// global key
const globalQuery = "query"
const globalResult = "result"
const globalError = "error"

type Swarm struct {
	// TODO experimental
	Root   string
	ChatID string

	Vars *api.Vars

	User *api.User

	Secrets api.SecretStore
	Assets  api.AssetManager
	Tools   api.ToolSystem

	Adapters llm.AdapterRegistry

	Blobs api.BlobStore

	OS        vos.System
	Workspace vfs.Workspace

	History api.MemStore

	// TODO
	template *template.Template
}

// https://pkg.go.dev/text/template
// https://masterminds.github.io/sprig/
func (r *Swarm) InitTemplate() {
	var fm = sprig.FuncMap()
	// overridge sprig
	fm["user"] = func() *api.User {
		return r.User
	}
	// OS
	getenv := func(key string) string {
		v, ok := r.Vars.Global.Get(key)
		if !ok {
			return ""
		}
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	fm["env"] = getenv
	fm["expandenv"] = func(s string) string {
		// bash name is leaked with os.Expand but ok.
		// bash is replaced with own that supports executing agent/tool
		return os.Expand(s, getenv)
	}
	// Network:
	fm["getHostByName"] = func() string {
		return "localhost"
	}

	// ai
	fm["ai"] = func(args ...string) string {
		var b bytes.Buffer
		ioe := &sh.IOE{Stdin: strings.NewReader(""), Stdout: &b, Stderr: &b}
		vs := sh.NewVirtualSystem(r.Root, r.OS, r.Workspace, ioe)
		var agent *api.Agent
		if v, ok := r.Vars.Global.Get("__parent_agent"); ok {
			agent = v.(*api.Agent)
		} else {
			return "Error: missing agent"
		}
		ctx := context.Background()
		result, err := r.runAgent(ctx, vs, agent, args)
		if err != nil {
			return err.Error()
		}
		if result == nil {
			return ""
		}
		return result.Value
	}

	r.template = template.New("swarm").Funcs(fm)
}

func (r *Swarm) createAgent(ctx context.Context, req *api.Request) (*api.Agent, error) {
	agent, err := conf.CreateAgent(ctx, req, r.User, r.Secrets, r.Assets)
	if err == nil && req.Parent != nil {
		r.Vars.Global.Set("__parent_agent", req.Parent)
	}
	return agent, err
}

// Run calls the language model with the messages list (after applying the system prompt). If the resulting AIMessage contains tool_calls, the graph will then call the tools. The tools node executes the tools and adds the responses to the messages list as ToolMessage objects. The agent node then calls the language model again. The process repeats until no more tool_calls are present in the response. The agent then returns the full list of messages.
func (r *Swarm) Run(req *api.Request, resp *api.Response) error {
	if req.Name == "" {
		return api.NewBadRequestError("missing agent in request")
	}
	if req.RawInput == nil {
		return api.NewBadRequestError("missing raw input in request")
	}

	if req.Parent != nil && req.Parent.Name == req.Name {
		return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself not supported.", req.Name))
	}

	var ctx = req.Context()
	var resetLogLevel = true

	if r.User == nil || r.Vars == nil {
		return api.NewInternalServerError("invalid config. user or vars not initialized")
	}

	//
	logMiddleware := MaxLogMiddlewareFunc(r)
	envMiddleware := EnvMiddlewareFunc(r)
	memMiddleware := MemoryMiddlewareFunc(r)

	instructMiddleware := InstructionMiddlewareFunc(r)
	QueryMiddleware := QueryMiddlewareFunc(r)
	contextMiddleware := ContextMiddlewareFunc(r)
	agentMiddlWare := AgentMiddlewareFunc(r)

	toolMiddleWare := ToolMiddlewareFunc(r)

	modelMiddleware := ModelMiddlewareFunc(r)

	inferMiddelWare := InferenceMiddlewareFunc(r)

	log.GetLogger(ctx).Debugf("*** Agent: %s parent: %+v\n", req.Name, req.Parent)

	final := HandlerFunc(func(req *api.Request, res *api.Response) error {
		log.GetLogger(ctx).Debugf("🔗 (final): %s\n", req.Name)
		return nil
	})

	for {
		start := time.Now()
		log.GetLogger(ctx).Debugf("Creating agent: %s %s\n", req.Name, start)
		//
		agent, err := r.createAgent(ctx, req)
		if err != nil {
			return err
		}

		// reset log level
		// subsequent agents will inhefit the same log level
		if resetLogLevel {
			resetLogLevel = false
			log.GetLogger(ctx).SetLogLevel(agent.LogLevel)
		}

		chain := NewChain(
			TimeoutMiddleware(agent),
			logMiddleware(agent, 100),
			//
			envMiddleware(agent),
			memMiddleware(agent),
			//
			instructMiddleware(agent),
			QueryMiddleware(agent),
			contextMiddleware(agent),
			agentMiddlWare(agent),
			toolMiddleWare(agent),
			modelMiddleware(agent),
			//
			inferMiddelWare(agent),
		)

		if err := chain.Then(final).Serve(req, resp); err != nil {
			return err
		}

		if resp.Result == nil {
			// some thing went wrong
			return fmt.Errorf("Empty result running %q", agent.Name)
		}

		if resp.Result.State == api.StateTransfer {
			log.GetLogger(ctx).Debugf("Agent transfer: %s => %s\n", req.Name, resp.Result.NextAgent)
			req.Name = resp.Result.NextAgent
			continue
		}

		end := time.Now()
		log.GetLogger(ctx).Debugf("Agent complete: %s %s elapsed: %s\n", req.Name, end, end.Sub(start))
		return nil
	}
}

// copy values from src to dst after calling @agent and applying template if required
func (r *Swarm) mapAssign(agent *api.Agent, req *api.Request, dst, src map[string]any, override bool) error {
	for key, val := range src {
		if !override {
			if _, ok := dst[key]; ok {
				continue
			}
		}
		// @agent value support
		if v, ok := val.(string); ok {
			if resolved, err := r.resolveArgument(agent, req, v); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		// go template value support
		if v, ok := val.(string); ok && strings.HasPrefix(v, "{{") {
			if resolved, err := r.applyTemplate(v, dst); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		dst[key] = val
	}
	return nil
}

// make a copy of golbal env
func (r *Swarm) globalEnv() map[string]any {
	var env = make(map[string]any)
	r.Vars.Global.Copy(env)
	return env
}

func (r *Swarm) RunSub(parent *api.Agent, req *api.Request, resp *api.Response) error {
	// prevent loop
	// TODO support recursion?
	if parent != nil && parent.Name == req.Name {
		return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself.", req.Name))
	}

	if err := r.Run(req, resp); err != nil {
		return err
	}
	if resp.Result == nil {
		return fmt.Errorf("Empty result")
	}

	return nil
}

// call agent if found. otherwise return s as is
func (r *Swarm) resolveArgument(agent *api.Agent, req *api.Request, s string) (any, error) {
	name, query, found := parseAgentCommand(s)
	if !found {
		return s, nil
	}
	out, err := r.callAgent(agent, req, name, query)
	if err != nil {
		return nil, err
	}

	type ArgResult struct {
		Result string
		Error  string
	}

	var arg ArgResult
	if err := json.Unmarshal([]byte(out), &arg); err != nil {
		return nil, err
	}
	if arg.Error != "" {
		return nil, fmt.Errorf("failed resolve argument: %s", arg.Error)
	}
	return arg.Result, nil
}

func (r *Swarm) callAgent(parent *api.Agent, req *api.Request, s string, prompt string) (string, error) {
	name := strings.TrimPrefix(s, "@")

	nreq := req.Clone()
	nreq.Parent = parent
	nreq.Name = name
	// prepend additional instruction to user query
	if len(prompt) > 0 {
		nreq.RawInput.Message = prompt + "\n" + nreq.RawInput.Message
	}

	nresp := &api.Response{}

	err := r.RunSub(parent, nreq, nresp)
	if err != nil {
		return "", err
	}
	if nresp.Result == nil {
		return "", fmt.Errorf("empty response")
	}
	return nresp.Result.Value, nil
}

func (r *Swarm) applyTemplate(text string, data any) (string, error) {
	t, err := r.template.Parse(text)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (sw *Swarm) runScript(ctx context.Context, parent *api.Agent, script string) (string, error) {
	var b bytes.Buffer
	ioe := &sh.IOE{Stdin: strings.NewReader(""), Stdout: &b, Stderr: &b}
	vs := sh.NewVirtualSystem(sw.Root, sw.OS, sw.Workspace, ioe)

	// set global env for bash script
	env := sw.globalEnv()

	for k, v := range env {
		vs.System.Setenv(k, v)
	}

	vs.ExecHandler = sw.newExecHandler(vs, parent)

	if err := vs.RunScript(ctx, script); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (sw *Swarm) newExecHandler(vs *sh.VirtualSystem, parent *api.Agent) sh.ExecHandler {
	return func(ctx context.Context, args []string) (bool, error) {
		if parent == nil {
			return true, fmt.Errorf("missing parent agent")
		}
		log.GetLogger(ctx).Debugf("parent: %s args: %+v\n", parent.Name, args)
		isAi := func(s string) bool {
			return s == "ai" || strings.HasPrefix(s, "@") || strings.HasPrefix(s, "/")
		}
		if isAi(strings.ToLower(args[0])) {
			log.GetLogger(ctx).Debugf("running ai agent/tool: %+v\n", args)

			_, err := sw.runAgent(ctx, vs, parent, args)
			if err != nil {
				return true, err
			}

			return true, nil
		}

		// internal list
		allowed := []string{"env", "printenv"}
		if slices.Contains(allowed, args[0]) {
			doBashCustom(vs, args)
			return true, nil
		}

		// bash core utils
		if did, err := sh.RunCoreUtils(ctx, vs, args); did {
			return did, err
		}

		// block other commands
		fmt.Fprintf(vs.IOE.Stderr, "command not supported: %s %+v\n", args[0], args[1:])
		return true, nil
	}
}

// run agent command line args.
func (sw *Swarm) runAgent(ctx context.Context, vs *sh.VirtualSystem, parent *api.Agent, args []string) (*api.Result, error) {
	runner := sw.agentRunner(vs, parent)
	return runner(ctx, args)
}

func (sw *Swarm) applyGlobal(ext, s string, env map[string]any) (string, error) {
	if strings.HasPrefix(s, "#!") {
		parts := strings.SplitN(s, "\n", 2)
		if len(parts) == 2 {
			// remove hashbang line
			return sw.applyTemplate(parts[1], env)
		}
		// remove hashbang
		return sw.applyTemplate(parts[0][2:], env)
	}
	if ext == "tpl" {
		return sw.applyTemplate(s, env)
	}
	return s, nil
}

// inherit parent tools including embedded agents
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

func (sw *Swarm) agentRunner(vs *sh.VirtualSystem, agent *api.Agent) func(context.Context, []string) (*api.Result, error) {
	var memo = sw.buildAgentToolMap(agent)

	return func(ctx context.Context, args []string) (*api.Result, error) {
		var owner string
		if agent != nil {
			owner = agent.Owner
		}
		at, err := conf.ParseAgentToolArgs(owner, args)
		if err != nil {
			return nil, err
		}

		var kit string
		var name string
		if at.Agent != nil {
			kit = "agent"
			name = nvl(at.Agent.Name, "anonymous")
		} else if at.Tool != nil {
			kit = at.Tool.Kit
			name = at.Tool.Name
		} else {
			return nil, fmt.Errorf("invalid ai command")
		}
		id := api.KitName(kit + ":" + name).ID()
		action, ok := memo[id]
		if !ok {
			return nil, fmt.Errorf("agent tool not declared for %s: %s", agent.Name, id)
		}

		vs.System.Setenv(globalQuery, at.Message)

		result, err := sw.doAction(ctx, agent, action, at.Arguments)
		if err != nil {
			vs.System.Setenv(globalError, err.Error())
			fmt.Fprintln(vs.IOE.Stderr, err.Error())
			return nil, err
		}

		fmt.Fprintln(vs.IOE.Stdout, result.Value)
		vs.System.Setenv(globalResult, result.Value)

		return result, nil
	}
}

func (sw *Swarm) createCaller(user *api.User, agent *api.Agent) api.ToolRunner {
	toolMap := sw.buildAgentToolMap(agent)

	return func(ctx context.Context, tid string, args map[string]any) (*api.Result, error) {
		v, ok := toolMap[tid]
		if !ok {
			return nil, fmt.Errorf("tool not found: %s", tid)
		}

		return sw.callTool(context.WithValue(ctx, api.SwarmUserContextKey, user), agent, v, args)
	}
}

func (sw *Swarm) createAICaller(agent *api.Agent) api.ToolRunner {
	return func(ctx context.Context, tid string, args map[string]any) (*api.Result, error) {
		tools, err := conf.LoadToolFunc(agent.Owner, tid, sw.Secrets, sw.Assets)
		if err != nil {
			return nil, err
		}
		for _, v := range tools {
			id := v.ID()
			if id == tid {
				return sw.callTool(ctx, agent, v, args)
			}
		}
		return nil, fmt.Errorf("invalid tool: %s", tid)
	}
}

func (sw *Swarm) callTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	log.GetLogger(ctx).Infof("⣿ %s:%s %+v\n", tf.Kit, tf.Name, formatArgs(args))

	result, err := sw.dispatch(ctx, agent, tf, args)

	if err != nil {
		log.GetLogger(ctx).Errorf("✗ error: %v\n", err)
	} else {
		log.GetLogger(ctx).Infof("✔ %s \n", head(result.String(), 180))
	}

	return result, err
}

func (sw *Swarm) callAgentType(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	// agent tool
	if tf.Kit == api.ToolTypeAgent {
		return sw.callAgentTool(ctx, agent, tf, args)
	}

	// ai tool
	if tf.Kit == "ai" {
		return sw.callAITool(ctx, agent, tf, args)
	}
	return nil, api.NewUnsupportedError("agent kit: " + tf.Kit)
}

func (sw *Swarm) callAgentTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	var req *api.Request
	msg, _ := atm.GetStrProp("message", args)
	input := &api.UserInput{
		Message: msg,
	}
	req = api.NewRequest(ctx, tf.Agent, input)
	req.Parent = agent
	req.Arguments = args

	resp := &api.Response{}

	err := sw.RunSub(agent, req, resp)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (sw *Swarm) callAITool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	aiKit := NewAIKit(sw, agent)
	return aiKit.Call(ctx, sw.Vars, "", tf, args)
}

// vars *api.Vars, agent *api.Agent,
func (sw *Swarm) dispatch(ctx context.Context, agent *api.Agent, v *api.ToolFunc, args map[string]any) (*api.Result, error) {
	// agent tool
	if v.Type == api.ToolTypeAgent {
		out, err := sw.callAgentType(ctx, agent, v, args)
		if err != nil {
			return nil, err
		}
		return sw.toResult(out), nil
	}

	// custom kits
	kit, err := sw.Tools.GetKit(v)
	if err != nil {
		return nil, err
	}

	env := &api.ToolEnv{
		Owner: agent.Owner,
	}
	out, err := kit.Call(ctx, sw.Vars, env, v, args)
	if err != nil {
		return nil, fmt.Errorf("failed to call function tool %s %s: %w", v.Kit, v.Name, err)
	}
	return sw.toResult(out), nil
}

func (sw *Swarm) toResult(v any) *api.Result {
	if r, ok := v.(*api.Result); ok {
		if len(r.Content) == 0 {
			return r
		}
		if r.MimeType == api.ContentTypeImageB64 {
			return r
		}
		if strings.HasPrefix(r.MimeType, "text/") {
			return &api.Result{
				MimeType: r.MimeType,
				Value:    string(r.Content),
			}
		}
		return &api.Result{
			MimeType: r.MimeType,
			Value:    dataURL(r.MimeType, r.Content),
		}
		// // image
		// // transform media response into data url
		// presigned, err := h.save(r)
		// if err != nil {
		// 	return &api.Result{
		// 		Value: err.Error(),
		// 	}
		// }

		// return &api.Result{
		// 	MimeType: r.MimeType,
		// 	Value:    presigned,
		// }
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

// flow actions
func (sw *Swarm) doAction(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (*api.Result, error) {
	env := sw.globalEnv()

	if len(args) > 0 {
		maps.Copy(env, args)
	}

	var runTool = sw.createCaller(sw.User, agent)
	result, err := runTool(ctx, tf.ID(), env)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("no result")
	}

	// TODO check states?
	sw.Vars.Global.Set(globalResult, result.Value)
	return result, nil
}

// // save and get the presigned url
// func (sw *Swarm) save(v *api.Result) (string, error) {
// 	id := NewBlobID()
// 	b := &api.Blob{
// 		ID:       id,
// 		MimeType: v.MimeType,
// 		Content:  v.Content,
// 	}
// 	err := sw.Blobs.Put(id, b)
// 	if err != nil {
// 		return "", err
// 	}
// 	return sw.Blobs.Presign(id)
// }
