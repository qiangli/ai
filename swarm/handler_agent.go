package swarm

import (
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func AgentMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {
	return func(agent *api.Agent) api.Middleware {
		return func(next Handler) Handler {
			var ah = &agentHandler{
				agent: agent,
				sw:    sw,
				next:  next,
			}
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				log.GetLogger(req.Context()).Debugf("🔗 (agent): %s flow: %+v\n", agent.Name, agent.Flow)

				return ah.Serve(req, resp)
			})
		}
	}
}

type agentHandler struct {
	agent *api.Agent
	sw    *Swarm

	next api.Handler
}

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
	h.sw.Vars.Global.Set(globalQuery, req.RawInput.Query())
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

	// run agent
	if h.agent.Instruction != nil && h.agent.Instruction.Content != "" {
		if err := h.doAgent(req, resp); err != nil {
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
		case api.FlowTypeReduce:
			if err := h.flowReduce(req, resp); err != nil {
				return err
			}
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
		h.sw.mapAssign(h.agent, req, env, h.agent.Environment, true)
	}

	// set agent and req defaults
	// set only when the key does not exist
	if h.agent.Arguments != nil {
		h.sw.mapAssign(h.agent, req, env, h.agent.Arguments, false)
	}

	h.sw.Vars.Global.Add(env)

	log.GetLogger(req.Context()).Debugf("global env: %+v\n", env)
	return nil
}

func (h *agentHandler) doAgent(req *api.Request, resp *api.Response) error {
	return h.next.Serve(req, resp)
}
