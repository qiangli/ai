package swarm

import (
	// "context"
	"encoding/json"
	"fmt"
	// "os"
	// "text/template"

	// "github.com/Masterminds/sprig/v3"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

func AgentMiddleware(sw *Swarm) api.Middleware {
	return func(agent *api.Agent, next Handler) Handler {
		var ah = &agentHandler{
			agent: agent,
			sw:    sw,
			next:  next,
		}
		// ah.initTemplate()
		// ah.initChain()
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			log.GetLogger(req.Context()).Debugf("🔗 (agent): %s flow: %+v\n", agent.Name, agent.Flow)

			return ah.Serve(req, resp)
		})
	}
}

type agentHandler struct {
	agent *api.Agent
	sw    *Swarm

	next api.Handler

	// template *template.Template

	// middlewares []api.Middleware
}

// // https://pkg.go.dev/text/template
// // https://masterminds.github.io/sprig/
// func (h *agentHandler) initTemplate() {
// 	sw := h.sw

// 	var fm = sprig.FuncMap()
// 	// overridge sprig
// 	fm["user"] = func() *api.User {
// 		return sw.User
// 	}
// 	// OS
// 	getenv := func(key string) string {
// 		env, ok := h.agent.Environment.Get(key)
// 		if !ok {
// 			v, ok := sw.Vars.Global.Get(key)
// 			if !ok {
// 				return ""
// 			} else {
// 				env = v
// 			}
// 		}
// 		if s, ok := env.(string); ok {
// 			return s
// 		}
// 		return fmt.Sprintf("%v", env)
// 	}
// 	fm["env"] = getenv
// 	fm["expandenv"] = func(s string) string {
// 		// bash name is leaked with os.Expand but ok.
// 		// bash is replaced with own that supports executing agent/tool
// 		return os.Expand(s, getenv)
// 	}
// 	// Network:
// 	fm["getHostByName"] = func() string {
// 		return "localhost"
// 	}

// 	// ai
// 	fm["ai"] = func(args ...string) string {
// 		at, err := conf.ParseActionArgs(args)
// 		if err != nil {
// 			return err.Error()
// 		}
// 		id := api.KitName(at.Name).ID()

// 		// var b bytes.Buffer
// 		// ioe := &sh.IOE{Stdin: strings.NewReader(""), Stdout: &b, Stderr: &b}
// 		// vs := sh.NewVirtualSystem(sw.Root, sw.OS, sw.Workspace, ioe)
// 		// var agent *api.Agent
// 		// if v, ok := sw.Vars.Global.GetAgent(at.Name); ok {
// 		// 	agent = v.(*api.Agent)
// 		// } else {
// 		// 	return fmt.Sprintf("Error: missing agent %q in env", at.Name)
// 		// }
// 		ctx := context.Background()
// 		// result, err := ExecAction(ctx, parent, args)
// 		// v, err := parent.Runner.Run(ctx, id, args)

// 		data, err := h.agent.Runner.Run(ctx, id, at.Arguments)
// 		if err != nil {
// 			// vs.System.Setenv(globalError, err.Error())
// 			// fmt.Fprintln(vs.IOE.Stderr, err.Error())
// 			return err.Error()
// 		}
// 		result := api.ToResult(data)
// 		if err != nil {
// 			return err.Error()
// 		}
// 		// result := api.ToResult(v)
// 		// if result == nil {
// 		// 	return ""
// 		// }
// 		return result.Value
// 	}

// 	h.template = template.New("swarm-agent").Funcs(fm)
// }

// func (h *agentHandler) initChain() {
// 	sw := h.sw
// 	h.middlewares = []api.Middleware{
// 		//input
// 		TimeoutMiddleware(sw),
// 		LogMiddleware(sw),
// 		EnvMiddleware(sw),
// 		MemoryMiddleware(sw),
// 		//
// 		InstructionMiddleware(sw),
// 		QueryMiddleware(sw),
// 		ContextMiddleware(sw),
// 		AgentMiddleware(sw),
// 		//
// 		ToolMiddleware(sw),
// 		//
// 		ModelMiddleware(sw),
// 		//
// 		InferenceMiddleware(sw),
// 		// metrics
// 		// output
// 	}
// }

// func (h *agentHandler) NewChain(ctx context.Context, a *api.Agent) api.Handler {
// 	log.GetLogger(ctx).Infof("🔗 (init): %s\n", a.Name)
// 	// var mds = make([]api.Middleware, len(sw.middlewares))
// 	// for i, v := range sw.middlewares {
// 	// 	mds[i] = v(a)
// 	// }
// 	final := HandlerFunc(func(req *api.Request, res *api.Response) error {
// 		log.GetLogger(req.Context()).Infof("🔗 (final): %s\n", req.Name)
// 		return nil
// 	})
// 	chain := NewChain(h.middlewares...).Then(a, final)
// 	return chain
// }

// Serve calls the language model adapter with the messages list (after applying the system prompt).
// If the resulting response contains tool_calls, the tool runner will then call the tools.
// The tools kit executes the tools and adds the responses to the messages list.
// The adapter then calls the language model again.
// The process repeats until no more tool_calls are present in the response.
// The agent handler then returns the full list of messages.
func (h *agentHandler) Serve(req *api.Request, resp *api.Response) error {
	var ctx = req.Context()
	log.GetLogger(ctx).Debugf("Serve agent: %s global: %+v\n", h.agent.Name, h.sw.Vars.Global)

	// this needs to happen before everything else
	// h.sw.Vars.Global.Set(globalQuery, req.RawInput.Query())
	h.sw.Vars.Global.Set(globalQuery, req.Query)

	if err := h.setGlobalEnv(req); err != nil {
		return err
	}

	if err := h.doAgentFlow(req, resp); err != nil {
		h.sw.Vars.Global.Set(globalResult, err.Error())
		return err
	}

	var result string
	if resp.Result != nil {
		result = resp.Result.Value
	}
	// if result is json, unpack for subsequnent agents/actions
	if len(result) > 0 {
		var resultMap = make(map[string]any)
		if err := json.Unmarshal([]byte(result), &resultMap); err == nil {
			h.sw.Vars.Global.Add(resultMap)
		}
	}
	h.sw.Vars.Global.Set(globalResult, result)

	log.GetLogger(ctx).Debugf("completed: %s global: %+v\n", h.agent.Name, h.sw.Vars.Global)
	return nil
}

// run agent first if there is instruction followed by the flow.
// otherwise, run the flow only
func (h *agentHandler) doAgentFlow(req *api.Request, resp *api.Response) error {
	if h.agent.Instruction == nil && h.agent.Flow == nil {
		return api.NewBadRequestError("missing instruction and flow")
	}

	// run llm inference
	if h.agent.Instruction != nil && h.agent.Instruction.Content != "" {
		if err := h.next.Serve(req, resp); err != nil {
			return err
		}
	}

	// flow control agent
	if h.agent.Flow != nil {
		if len(h.agent.Flow.Actions) == 0 && len(h.agent.Flow.Script) == 0 {
			return fmt.Errorf("missing actions or script in flow")
		}
		switch h.agent.Flow.Type {
		case api.FlowTypeSequence:
			if err := h.flowSequence(req, resp); err != nil {
				return err
			}
		case api.FlowTypeParallel:
			if err := h.flowParallel(req, resp); err != nil {
				return err
			}
		case api.FlowTypeChoice:
			if err := h.flowChoice(req, resp); err != nil {
				return err
			}
		case api.FlowTypeMap:
			if err := h.flowMap(req, resp); err != nil {
				return err
			}
		case api.FlowTypeLoop:
			if err := h.flowLoop(req, resp); err != nil {
				return err
			}
		// case api.FlowTypeReduce:
		// 	if err := h.flowReduce(req, resp); err != nil {
		// 		return err
		// 	}
		case api.FlowTypeShell:
			if err := h.flowShell(req, resp); err != nil {
				return err
			}
		default:
			return fmt.Errorf("not supported yet %v", h.agent.Flow)
		}
	}

	return nil
}

// create a copy of current global vars
// merge agent environment, update with values from agent arguments if non existant
// support @agent call and go template as value
func (h *agentHandler) setGlobalEnv(req *api.Request) error {
	var env = make(map[string]any)
	// copy globals including agent args
	h.sw.Vars.Global.Copy(env)

	// agent global env takes precedence
	if h.agent.Environment != nil {
		h.sw.mapAssign(h.agent, req, env, h.agent.Environment.GetEnvs(nil), true)
	}

	// set agent and req defaults
	// set only when the key does not exist
	if h.agent.Arguments != nil {
		h.sw.mapAssign(h.agent, req, env, h.agent.Arguments, false)
	}

	// h.sw.Vars.Global.Add(env)

	log.GetLogger(req.Context()).Debugf("global env: %+v\n", env)
	return nil
}

// func (h *agentHandler) doNext(req *api.Request, resp *api.Response) error {
// 	return h.next.Serve(req, resp)
// }
