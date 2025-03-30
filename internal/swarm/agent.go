package swarm

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm/llm"
)

const transferAgentName = "agent_transfer"

type Agent struct {
	// The name of the agent.
	Name string

	Display string

	// The model to be used by the agent
	Model *api.Model

	// The role of the agent. default is "system"
	Role string

	// Instructions for the agent, can be a string or a callable returning a string
	Instruction string

	RawInput *api.UserInput

	// Vars *Vars

	// Functions that the agent can call
	Tools []*api.ToolFunc

	Entrypoint api.Entrypoint

	Dependencies []*Agent

	// advices
	BeforeAdvice api.Advice
	AfterAdvice  api.Advice
	AroundAdvice api.Advice

	//
	MaxTurns int
	MaxTime  int

	//
	sw *Swarm
}

func (r *Agent) Vars() *api.Vars {
	return r.sw.Vars
}

// type Agent struct {
// 	api.Agent

// 	sw *Swarm
// }
// 	// The name of the agent.

func (r *Agent) Serve(req *api.Request, resp *api.Response) error {
	log.Debugf("run agent: %s\n", r.Name)

	ctx := req.Context()

	// dependencies
	if len(r.Dependencies) > 0 {
		for _, dep := range r.Dependencies {
			depReq := &api.Request{
				Agent:    dep.Name,
				RawInput: req.RawInput,
				Message:  req.Message,
			}
			depResp := &api.Response{}
			if err := r.sw.Run(depReq, depResp); err != nil {
				return err
			}
			log.Debugf("run dependency: %v %+v\n", dep.Display, depResp)
		}
	}

	// advices
	noop := func(vars *api.Vars, _ *api.Request, _ *api.Response, _ api.Advice) error {
		return nil
	}
	if r.BeforeAdvice != nil {
		if err := r.BeforeAdvice(r.sw.Vars, req, resp, noop); err != nil {
			return err
		}
	}
	if r.AroundAdvice != nil {
		next := func(vars *api.Vars, req *api.Request, resp *api.Response, _ api.Advice) error {
			return r.runLoop(ctx, req, resp)
		}
		if err := r.AroundAdvice(r.sw.Vars, req, resp, next); err != nil {
			return err
		}
	} else {
		if err := r.runLoop(ctx, req, resp); err != nil {
			return err
		}
	}
	if r.AfterAdvice != nil {
		if err := r.AfterAdvice(r.sw.Vars, req, resp, noop); err != nil {
			return err
		}
	}

	return nil
}

func (r *Agent) runLoop(ctx context.Context, req *api.Request, resp *api.Response) error {
	// "resource:" prefix is used to refer to a resource
	// "vars:" prefix is used to refer to a variable
	apply := func(s string, vars *api.Vars) (string, error) {
		if strings.HasPrefix(s, "resource:") {
			v, ok := r.sw.Config.ResourceMap[s[9:]]
			if !ok {
				return "", fmt.Errorf("no such resource: %s", s[9:])
			}
			return applyTemplate(v, vars, r.sw.Config.TemplateFuncMap)
		}
		if strings.HasPrefix(s, "vars:") {
			v := vars.GetString(s[5:])
			return v, nil
		}
		return s, nil
	}

	var history []*api.Message

	// system role prompt as first message
	if r.Instruction != "" {
		// update the request instruction
		content, err := apply(r.Instruction, r.sw.Vars)
		if err != nil {
			return err
		}

		role := r.Role
		if role == "" {
			role = api.RoleSystem
		}
		history = append(history, &api.Message{
			Role:    role,
			Content: content,
			Sender:  r.Name,
		})
	}
	// FIXME: this is confusing LLM?
	// history = append(history, r.sw.History...)

	if req.Message == nil {
		req.Message = &api.Message{
			Role:    api.RoleUser,
			Content: req.RawInput.Query(),
			Sender:  r.Name,
		}
	}
	history = append(history, req.Message)

	initLen := len(history)
	agentRole := r.Role
	if agentRole == "" {
		agentRole = api.RoleAssistant
	}

	runTool := func(ctx context.Context, name string, args map[string]any) (*api.Result, error) {
		log.Debugf("run tool: %s %+v\n", name, args)
		return CallTool(ctx, r.sw.Vars, name, args)
	}

	result, err := llm.Send(ctx, &api.LLMRequest{
		Agent:     r.Name,
		ModelType: r.Model.Type,
		BaseUrl:   r.Model.BaseUrl,
		ApiKey:    r.Model.ApiKey,
		Model:     r.Model.Name,
		Messages:  history,
		MaxTurns:  r.MaxTurns,
		RunTool:   runTool,
		Tools:     r.Tools,
		//
		ImageQuality: req.ImageQuality,
		ImageSize:    req.ImageSize,
		ImageStyle:   req.ImageStyle,
	})
	if err != nil {
		return err
	}

	if result.Result == nil || result.Result.State != api.StateTransfer {
		message := api.Message{
			ContentType: result.ContentType,
			Role:        result.Role,
			Content:     result.Content,
			Sender:      r.Name,
		}
		history = append(history, &message)
	}

	resp.Messages = history[initLen:]

	resp.Agent = &api.Agent{
		Name:    r.Name,
		Display: r.Display,
	}
	resp.Result = result.Result

	r.sw.History = history
	return nil
}
