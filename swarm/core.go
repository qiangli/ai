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

	middlewares []api.Middleware

	agentMaker *AgentMaker
}

func (sw *Swarm) Init() {
	sw.InitTemplate()
	sw.InitChain()
	sw.agentMaker = NewAgentMaker(sw)
}

// https://pkg.go.dev/text/template
// https://masterminds.github.io/sprig/
func (sw *Swarm) InitTemplate() {
	var fm = sprig.FuncMap()
	// overridge sprig
	fm["user"] = func() *api.User {
		return sw.User
	}
	// OS
	getenv := func(key string) string {
		v, ok := sw.Vars.Global.Get(key)
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
		vs := sh.NewVirtualSystem(sw.Root, sw.OS, sw.Workspace, ioe)
		var parent *api.Agent
		if v, ok := sw.Vars.Global.Get("__parent_agent"); ok {
			parent = v.(*api.Agent)
		} else {
			return "Error: missing agent"
		}
		ctx := context.Background()
		result, err := sw.runAgent(ctx, vs, parent, args)
		// result, err := sw.doAction(ctx, vs, parent, args)

		if err != nil {
			return err.Error()
		}
		if result == nil {
			return ""
		}
		return result.Value
	}

	sw.template = template.New("swarm").Funcs(fm)
}

func (sw *Swarm) InitChain() {
	sw.middlewares = []api.Middleware{
		//input
		TimeoutMiddleware(sw),
		MaxLogMiddleware(sw),
		EnvMiddleware(sw),
		MemoryMiddleware(sw),
		//
		InstructionMiddleware(sw),
		QueryMiddleware(sw),
		ContextMiddleware(sw),
		AgentMiddleware(sw),
		//
		ToolMiddleware(sw),
		//
		ModelMiddleware(sw),
		//
		InferenceMiddleware(sw),
		// output
	}
}

func (sw *Swarm) NewChain(ctx context.Context, a *api.Agent) api.Handler {
	log.GetLogger(ctx).Infof("🔗 (init): %s\n", a.Name)
	// var mds = make([]api.Middleware, len(sw.middlewares))
	// for i, v := range sw.middlewares {
	// 	mds[i] = v(a)
	// }
	final := HandlerFunc(func(req *api.Request, res *api.Response) error {
		log.GetLogger(req.Context()).Infof("🔗 (final): %s\n", req.Name)
		return nil
	})
	chain := NewChain(sw.middlewares...).Then(a, final)
	return chain
}

func (sw *Swarm) createAgent(ctx context.Context, req *api.Request) (*api.Agent, error) {
	// agent, err := conf.CreateAgent(ctx, req, sw.User, sw.Secrets, sw.Assets)
	agent, err := sw.agentMaker.CreateAgent(ctx, req.Name)

	if err != nil {
		return nil, err
	}

	agent.Runner = sw.createCaller(agent)
	if req.Parent != nil {
		sw.Vars.Global.Set("__parent_agent", req.Parent)
	}
	return agent, nil
}

// Run calls the language model with the messages list (after applying the system prompt).
// If the resulting AI Message contains tool_calls, the orchestrator will then call the tools.
// The tools node executes the tools and adds the responses to the messages list as ToolMessage objects. The agent node then calls the language model again. The process repeats until no more tool_calls are present in the response. The agent then returns the full list of messages.
func (sw *Swarm) Run(req *api.Request, resp *api.Response) error {

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
	logger := log.GetLogger(ctx)

	setLogLevel := func(a *api.Agent) {
		ll := a.LogLevel
		for {
			if a.Parent == nil {
				break
			}
			a = a.Parent
			ll = a.LogLevel
		}
		logger.SetLogLevel(ll)
	}

	if sw.User == nil || sw.Vars == nil {
		return api.NewInternalServerError("invalid config. user or vars not initialized")
	}

	for {
		start := time.Now()
		logger.Debugf("creating agent: %s %s\n", req.Name, start)

		// creator
		agent, err := sw.createAgent(ctx, req)
		if err != nil {
			return err
		}

		setLogLevel(agent)

		logger.Infof("🚀 %s ← %s\n", agent.Name, NilSafe(agent.Parent).Name)

		// init
		if err := sw.NewChain(ctx, agent).Serve(req, resp); err != nil {
			return err
		}

		if resp.Result == nil {
			// some thing went wrong
			return fmt.Errorf("Empty result running %q", agent.Name)
		}

		if resp.Result.State == api.StateTransfer {
			logger.Debugf("Agent transfer: %s => %s\n", req.Name, resp.Result.NextAgent)
			req.Name = resp.Result.NextAgent
			continue
		}

		end := time.Now()
		logger.Debugf("Agent complete: %s %s elapsed: %s\n", req.Name, end, end.Sub(start))
		return nil
	}
}

// copy values from src to dst after calling @agent and applying template if required
func (sw *Swarm) mapAssign(agent *api.Agent, req *api.Request, dst, src map[string]any, override bool) error {
	for key, val := range src {
		if !override {
			if _, ok := dst[key]; ok {
				continue
			}
		}
		// @agent value support
		if v, ok := val.(string); ok {
			if resolved, err := sw.resolveArgument(agent, req, v); err != nil {
				return err
			} else {
				val = resolved
			}
		}
		// go template value support
		if v, ok := val.(string); ok && strings.HasPrefix(v, "{{") {
			if resolved, err := sw.applyTemplate(v, dst); err != nil {
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
func (sw *Swarm) globalEnv() map[string]any {
	var env = make(map[string]any)
	sw.Vars.Global.Copy(env)
	return env
}

func (sw *Swarm) RunSub(parent *api.Agent, req *api.Request, resp *api.Response) error {
	// prevent loop
	// TODO support recursion?
	if parent != nil && parent.Name == req.Name {
		return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself.", req.Name))
	}

	if err := sw.Run(req, resp); err != nil {
		return err
	}
	if resp.Result == nil {
		return fmt.Errorf("Empty result")
	}

	return nil
}

// call agent if found. otherwise return s as is
func (sw *Swarm) resolveArgument(agent *api.Agent, req *api.Request, s string) (any, error) {
	at, found := parseAgentCommand(s)
	if !found {
		return s, nil
	}
	out, err := sw.callAgent(agent, req, at.Name, at.Message)
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

func (sw *Swarm) callAgent(parent *api.Agent, req *api.Request, name string, message string) (string, error) {
	req.Parent = parent
	req.Name = strings.TrimPrefix(name, "@")
	// prepend additional instruction to user query
	req.Query = concat('\n', message, req.Query)

	resp := &api.Response{}

	err := sw.RunSub(parent, req, resp)
	if err != nil {
		return "", err
	}
	if resp.Result == nil || resp.Result.Value == "" {
		return "", fmt.Errorf("empty response")
	}
	return resp.Result.Value, nil
}

func (sw *Swarm) applyTemplate(text string, data any) (string, error) {
	t, err := sw.template.Parse(text)
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
		at, err := conf.ParseArgs(args)
		if err != nil {
			return nil, err
		}

		id := api.KitName(at.Name).ID()
		action, ok := memo[id]
		if !ok {
			return nil, fmt.Errorf("agent tool not declared for %s: %s", agent.Name, id)
		}

		vs.System.Setenv(globalQuery, at.Message)

		result, err := sw.doAction(ctx, agent, action.ID(), at.Arguments)
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

type AgentRunner struct {
	sw      *Swarm
	agent   *api.Agent
	toolMap map[string]*api.ToolFunc
}

func NewAgentRunner(sw *Swarm, agent *api.Agent) api.ActionRunner {
	toolMap := sw.buildAgentToolMap(agent)
	return &AgentRunner{
		sw:      sw,
		agent:   agent,
		toolMap: toolMap,
	}
}

func (r *AgentRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
	v, ok := r.toolMap[tid]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", tid)
	}

	result, err := r.sw.callTool(context.WithValue(ctx, api.SwarmUserContextKey, r.sw.User), r.agent, v, args)
	// log calls
	r.sw.Vars.AddToolCall(&api.ToolCallEntry{
		ID:        tid,
		Kit:       v.Kit,
		Name:      v.Name,
		Arguments: v.Arguments,
		Result:    result,
		Error:     err,
		Timestamp: time.Now(),
	})
	return result, err
}

func (sw *Swarm) createCaller(agent *api.Agent) api.ActionRunner {
	// toolMap := sw.buildAgentToolMap(agent)

	// return func(ctx context.Context, tid string, args map[string]any) (any, error) {
	// 	v, ok := toolMap[tid]
	// 	if !ok {
	// 		return nil, fmt.Errorf("tool not found: %s", tid)
	// 	}

	// 	result, err := sw.callTool(context.WithValue(ctx, api.SwarmUserContextKey, sw.User), agent, v, args)
	// 	// log calls
	// 	sw.Vars.AddToolCall(&api.ToolCallEntry{
	// 		ID:        tid,
	// 		Kit:       v.Kit,
	// 		Name:      v.Name,
	// 		Arguments: v.Arguments,
	// 		Result:    result,
	// 		Error:     err,
	// 		Timestamp: time.Now(),
	// 	})
	// 	return result, err
	// }
	return NewAgentRunner(sw, agent)
}

type AIAgentRunner struct {
	sw    *Swarm
	agent *api.Agent
}

func NewAIAgentRunner(sw *Swarm, agent *api.Agent) api.ActionRunner {
	return &AIAgentRunner{
		sw:    sw,
		agent: agent,
	}
}
func (r *AIAgentRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
	tools, err := conf.LoadToolFunc(r.agent.Owner, tid, r.sw.Secrets, r.sw.Assets)
	if err != nil {
		return nil, err
	}
	for _, v := range tools {
		id := v.ID()
		if id == tid {
			return r.sw.callTool(ctx, r.agent, v, args)
		}
	}
	return nil, fmt.Errorf("invalid tool: %s", tid)
}

func (sw *Swarm) createAICaller(agent *api.Agent) api.ActionRunner {
	// return func(ctx context.Context, tid string, args map[string]any) (any, error) {
	// 	tools, err := conf.LoadToolFunc(agent.Owner, tid, sw.Secrets, sw.Assets)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	for _, v := range tools {
	// 		id := v.ID()
	// 		if id == tid {
	// 			return sw.callTool(ctx, agent, v, args)
	// 		}
	// 	}
	// 	return nil, fmt.Errorf("invalid tool: %s", tid)
	// }
	return NewAIAgentRunner(sw, agent)
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
		return ToResult(out), nil
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
	return ToResult(out), nil
}

func ToResult(data any) *api.Result {
	if v, ok := data.(*api.Result); ok {
		if len(v.Content) == 0 {
			return v
		}
		if v.MimeType == api.ContentTypeImageB64 {
			return v
		}
		if strings.HasPrefix(v.MimeType, "text/") {
			return &api.Result{
				MimeType: v.MimeType,
				Value:    string(v.Content),
			}
		}
		return &api.Result{
			MimeType: v.MimeType,
			Value:    dataURL(v.MimeType, v.Content),
		}
		// // image
		// // transform media response into data url
		// presigned, err := sw.save(sw)
		// if err != nil {
		// 	return &api.Result{
		// 		Value: err.Error(),
		// 	}
		// }

		// return &api.Result{
		// 	MimeType: v.MimeType,
		// 	Value:    presigned,
		// }
	}
	if s, ok := data.(string); ok {
		return &api.Result{
			Value: s,
		}
	}
	return &api.Result{
		Value: fmt.Sprintf("%v", data),
	}
}

// flow actions
func (sw *Swarm) doAction(ctx context.Context, agent *api.Agent, id string, args map[string]any) (*api.Result, error) {
	env := sw.globalEnv()

	if len(args) > 0 {
		maps.Copy(env, args)
	}

	// var runTool = sw.createCaller(sw.User, agent)
	var runTool = agent.Runner
	v, err := runTool.Run(ctx, id, env)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, fmt.Errorf("no result")
	}

	// TODO check states?
	result := ToResult(v)
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
