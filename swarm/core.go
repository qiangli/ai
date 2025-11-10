package swarm

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/llm"
	"github.com/qiangli/ai/swarm/log"
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

func (r *Swarm) createAgent(ctx context.Context, req *api.Request) (*api.Agent, error) {
	return conf.CreateAgent(ctx, req, r.User, r.Secrets, r.Assets)
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

	var fm = sprig.FuncMap()
	maps.Copy(fm, tplFuncMap)
	fm["user"] = func() *api.User {
		return r.User
	}
	r.template = template.New("swarm").Funcs(fm)

	logMiddleware := MaxLogMiddlewareFunc(r)
	envMiddleware := EnvMiddlewareFunc(r)
	memMiddleware := MemoryMiddlewareFunc(r)
	modelMiddleware := ModelMiddlewareFunc(r)
	agentMiddlWare := AgentMiddlewareFunc(r)
	contextMiddleware := ContextMiddlewareFunc(r)
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
			modelMiddleware(agent),
			agentMiddlWare(agent),
			contextMiddleware(agent),
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
			if resolved, err := applyTemplate(r.template, v, dst); err != nil {
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

// dynamcally make LLM model; return s as is if not an agent command
func (r *Swarm) resolveModel(parent *api.Agent, ctx context.Context, req *api.Request, m *api.Model) (*api.Model, error) {
	if m == nil {
		return nil, fmt.Errorf("missling model")
	}
	agent, query, found := parseAgentCommand(m.Model)
	if !found {
		return m, nil
	}
	out, err := r.callAgent(parent, req, agent, query)
	if err != nil {
		return nil, err
	}
	var model api.Model
	if err := json.Unmarshal([]byte(out), &model); err != nil {
		return nil, err
	}

	log.GetLogger(ctx).Infof("🤖 model: %s/%s\n", model.Provider, model.Model)

	// // replace api key
	// ak, err := h.sw.Secrets.Get(h.sw.User.Email, model.ApiKey)
	// if err != nil {
	// 	return nil, err
	// }
	// model.ApiKey = ak
	return &model, nil
}

// dynamically generate prompt if content starts with @<agent>
// otherwise, return s unchanged
func (r *Swarm) resolvePrompt(parent *api.Agent, req *api.Request, s string) (string, error) {
	name, query, found := parseAgentCommand(s)
	if !found {
		return s, nil
	}
	prompt, err := r.callAgent(parent, req, name, query)
	if err != nil {
		return "", err
	}

	// log.GetLogger(ctx).Infof("🤖 prompt: %s\n", head(prompt, 100))

	return prompt, nil
}

func (r *Swarm) mustResolveContext(parent *api.Agent, req *api.Request, s string) ([]*api.Message, error) {
	name, query, found := parseAgentCommand(s)
	if !found {
		return nil, fmt.Errorf("invalid context: %s", s)
	}
	out, err := r.callAgent(parent, req, name, query)
	if err != nil {
		return nil, err
	}

	var list []*api.Message
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil, err
	}

	// log.GetLogger(ctx).Debugf("dynamic context messages: (%v) %s\n", len(list), head(out, 100))
	return list, nil
}
