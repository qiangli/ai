package swarm

import (
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
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
	// h.sw.Vars.Global.Set(globalQuery, req.Query())

	if err := h.doAgentFlow(req, resp); err != nil {
		h.sw.Vars.Global.Set(globalError, err.Error())
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
			h.sw.Vars.Global.AddEnvs(resultMap)
		}
	}
	h.sw.Vars.Global.Set(globalResult, result)

	log.GetLogger(ctx).Debugf("completed: %s global: %+v\n", h.agent.Name, h.sw.Vars.Global)
	return nil
}

// run agent first if there is instruction followed by the flow.
// otherwise, run the flow only
func (h *agentHandler) doAgentFlow(req *api.Request, resp *api.Response) error {
	instruction := h.agent.Instruction()
	if instruction == "" && h.agent.Flow == nil {
		// no op?
		return api.NewBadRequestError("missing instruction and flow")
	}

	// run llm inference
	if instruction != "" {
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
