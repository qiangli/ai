package swarm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"

	"github.com/qiangli/ai/swarm/api"
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
	// swarm session id
	ChatID string

	Vars *api.Vars

	User *api.User

	Secrets api.SecretStore

	Assets api.AssetManager

	Tools api.ToolSystem

	Adapters llm.AdapterRegistry

	Blobs api.BlobStore

	// virtual system
	Root      string
	OS        vos.System
	Workspace vfs.Workspace

	History api.MemStore

	// internal
	middlewares []api.Middleware
	agentMaker  *AgentMaker
}

func (sw *Swarm) Init() {
	// sw.InitTemplate()
	sw.InitChain()
	sw.agentMaker = NewAgentMaker(sw)
}

// https://pkg.go.dev/text/template
// https://masterminds.github.io/sprig/
func (sw *Swarm) InitTemplate(agent *api.Agent) *template.Template {
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
		at, err := conf.ParseActionArgs(args)
		if err != nil {
			return err.Error()
		}
		id := api.KitName(at.Name).ID()

		ctx := context.Background()
		// result, err := ExecAction(ctx, parent, args)
		// v, err := parent.Runner.Run(ctx, id, args)

		data, err := agent.Runner.Run(ctx, id, at.Arguments)
		if err != nil {
			// vs.System.Setenv(globalError, err.Error())
			// fmt.Fprintln(vs.IOE.Stderr, err.Error())
			return err.Error()
		}
		result := api.ToResult(data)
		if err != nil {
			return err.Error()
		}

		return result.Value
	}

	return template.New("swarm").Funcs(fm)
}

func (sw *Swarm) InitChain() {

	// logging, analytics, and debugging.
	// prompts, tool selection, and output formatting.
	// retries, fallbacks, early termination.
	// rate limits, guardrails, pii detection.
	sw.middlewares = []api.Middleware{
		InitEnvMiddleware(sw),
		//
		TimeoutMiddleware(sw),
		LogMiddleware(sw),
		//
		ModelMiddleware(sw),
		ToolMiddleware(sw),
		//
		MemoryMiddleware(sw),
		InstructionMiddleware(sw),
		QueryMiddleware(sw),
		ContextMiddleware(sw),
		//
		AgentMiddleware(sw),
		//
		InferenceMiddleware(sw),
		// output
	}
}

func (sw *Swarm) NewChain(ctx context.Context, a *api.Agent) api.Handler {
	final := HandlerFunc(func(req *api.Request, res *api.Response) error {
		log.GetLogger(req.Context()).Infof("🔗 (final): %s\n", req.Name)
		return nil
	})

	chain := NewChain(sw.middlewares...).Then(a, final)
	return chain
}

func (sw *Swarm) createAgent(ctx context.Context, req *api.Request) (*api.Agent, error) {
	var name = req.Name
	if name == "" {
		// anonymous
		log.GetLogger(ctx).Debugf("agent not specified.\n")
	}
	agent, err := sw.agentMaker.CreateAgent(ctx, name)

	if err != nil {
		return nil, err
	}

	// for sub agent/action or tool call
	agent.Runner = NewAgentToolRunner(sw, agent)
	agent.Template = sw.InitTemplate(agent)
	return agent, nil
}

// Run calls the language model with the messages list (after applying the system prompt).
// If the resulting AI Message contains tool_calls, the orchestrator will then call the tools.
// The tools node executes the tools and adds the responses to the messages list as ToolMessage objects. The agent node then calls the language model again. The process repeats until no more tool_calls are present in the response. The agent then returns the full list of messages.
func (sw *Swarm) Run(req *api.Request, resp *api.Response) error {
	if sw.User == nil || sw.Vars == nil {
		return api.NewInternalServerError("invalid config. user or vars not initialized")
	}

	if req.Parent != nil && req.Parent.Name == req.Name {
		return api.NewUnsupportedError(fmt.Sprintf("agent: %q calling itself not supported.", req.Name))
	}

	var ctx = req.Context()
	logger := log.GetLogger(ctx)

	for {
		start := time.Now()
		logger.Debugf("creating agent: %s %s\n", req.Name, start)

		// creator
		agent, err := sw.createAgent(ctx, req)
		if err != nil {
			return err
		}

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
			if resolved, err := applyTemplate(agent.Template, v, dst); err != nil {
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
	req.Name = strings.TrimPrefix(name, "agent:")
	// prepend additional instruction to user query
	// req.SetQuery(concat('\n', message, req.Query()))

	resp := &api.Response{}

	err := sw.RunSub(parent, req, resp)
	if err != nil {
		return "", err
	}
	if resp.Result == nil {
		return "", nil
	}
	return resp.Result.Value, nil
}

func applyTemplate(tpl *template.Template, text string, data any) (string, error) {
	t, err := tpl.Parse(text)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type AgentScriptRunner struct {
	sw    *Swarm
	agent *api.Agent
}

func NewAgentScriptRunner(sw *Swarm, agent *api.Agent) api.ActionRunner {
	return &AgentScriptRunner{
		sw:    sw,
		agent: agent,
	}
}

func (r *AgentScriptRunner) Run(ctx context.Context, script string, args map[string]any) (any, error) {
	var b bytes.Buffer
	ioe := &sh.IOE{Stdin: strings.NewReader(""), Stdout: &b, Stderr: &b}
	vs := sh.NewVirtualSystem(r.sw.Root, r.sw.OS, r.sw.Workspace, ioe)

	// set global env for bash script
	env := r.sw.globalEnv()

	for k, v := range env {
		vs.System.Setenv(k, v)
	}

	vs.ExecHandler = r.newExecHandler(vs, r.agent)

	if err := vs.RunScript(ctx, script); err != nil {
		return "", err
	}

	return b.String(), nil
}

func (r *AgentScriptRunner) newExecHandler(vs *sh.VirtualSystem, parent *api.Agent) sh.ExecHandler {
	runner := r.runner(vs, parent)
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

			_, err := runner(ctx, args)
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

func applyGlobal(tpl *template.Template, ext, s string, env map[string]any) (string, error) {
	if strings.HasPrefix(s, "#!") {
		// TODO parse the command line args?
		parts := strings.SplitN(s, "\n", 2)
		if len(parts) == 2 {
			// remove hashbang line
			return applyTemplate(tpl, parts[1], env)
		}
		// remove hashbang
		return applyTemplate(tpl, parts[0][2:], env)
	}
	if strings.HasPrefix(s, "{{") && strings.HasSuffix(s, "}}") {
		return applyTemplate(tpl, s, env)
	}
	if ext == "tpl" {
		return applyTemplate(tpl, s, env)
	}
	return s, nil
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

func (r *AgentScriptRunner) runner(vs *sh.VirtualSystem, agent *api.Agent) func(context.Context, []string) (*api.Result, error) {
	// var memo = r.sw.buildAgentToolMap(agent)

	return func(ctx context.Context, args []string) (*api.Result, error) {
		at, err := conf.ParseActionArgs(args)
		if err != nil {
			return nil, err
		}
		id := api.KitName(at.Name).ID()
		// action, ok := memo[id]
		// if !ok {
		// 	return nil, fmt.Errorf("agent tool not declared for %s: %s", agent.Name, id)
		// }

		vs.System.Setenv(globalQuery, at.Message)

		// result, err := r.sw.RunAction(ctx, agent, action.ID(), at.Arguments)
		data, err := agent.Runner.Run(ctx, id, at.Arguments)
		if err != nil {
			vs.System.Setenv(globalError, err.Error())
			fmt.Fprintln(vs.IOE.Stderr, err.Error())
			return nil, err
		}
		result := api.ToResult(data)

		fmt.Fprintln(vs.IOE.Stdout, result.Value)
		vs.System.Setenv(globalResult, result.Value)

		return result, nil
	}
}

type AgentToolRunner struct {
	sw      *Swarm
	agent   *api.Agent
	toolMap map[string]*api.ToolFunc
}

func NewAgentToolRunner(sw *Swarm, agent *api.Agent) api.ActionRunner {
	toolMap := sw.buildAgentToolMap(agent)
	return &AgentToolRunner{
		sw:      sw,
		agent:   agent,
		toolMap: toolMap,
	}
}

func (r *AgentToolRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
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

type AIAgentToolRunner struct {
	sw    *Swarm
	agent *api.Agent
}

func NewAIAgentToolRunner(sw *Swarm, agent *api.Agent) api.ActionRunner {
	return &AIAgentToolRunner{
		sw:    sw,
		agent: agent,
	}
}

func (r *AIAgentToolRunner) Run(ctx context.Context, tid string, args map[string]any) (any, error) {
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
		return sw.callAIAgentTool(ctx, agent, tf, args)
	}
	return nil, api.NewUnsupportedError("agent kit: " + tf.Kit)
}

func (sw *Swarm) callAgentTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	// var req *api.Request
	// msg, _ := atm.GetStrProp("message", args)
	input := &api.UserInput{
		// Message:   msg,
		Arguments: args,
	}
	req := api.NewRequest(ctx, tf.Agent, input)
	req.Parent = agent

	resp := &api.Response{}

	err := sw.RunSub(agent, req, resp)
	if err != nil {
		return nil, err
	}

	return resp.Result, nil
}

func (sw *Swarm) callAIAgentTool(ctx context.Context, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
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
