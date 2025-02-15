package swarm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
)

const transferAgentName = "agent_transfer"

type Agent struct {
	// The name of the agent.
	Name string

	Display string

	// The model to be used by the agent
	Model *Model

	// The role of the agent. default is "system"
	Role string

	// Instructions for the agent, can be a string or a callable returning a string
	Instruction string

	Vars *Vars

	// A list of functions that the agent can call
	Functions []*ToolFunc

	Entrypoint Entrypoint

	Dependencies []*Agent

	// advices
	BeforeAdvice Advice
	AfterAdvice  Advice
	AroundAdvice Advice

	//
	MaxTurns int
	MaxTime  int

	//
	sw *Swarm
}

func (r *Agent) Serve(req *Request, resp *Response) error {
	log.Debugf("run agent: %s\n", r.Name)

	ctx := req.Context()

	// dependencies
	if len(r.Dependencies) > 0 {
		for _, dep := range r.Dependencies {
			depReq := &Request{
				Agent:    dep.Name,
				RawInput: req.RawInput,
				Message:  req.Message,
			}
			depResp := &Response{}
			if err := r.sw.Run(depReq, depResp); err != nil {
				return err
			}
			log.Debugf("run dependency: %v %+v\n", dep.Display, depResp)
		}
	}

	// advices
	noop := func(vars *Vars, _ *Request, _ *Response, _ Advice) error {
		return nil
	}
	if r.BeforeAdvice != nil {
		if err := r.BeforeAdvice(r.Vars, req, resp, noop); err != nil {
			return err
		}
	}
	if r.AroundAdvice != nil {
		next := func(vars *Vars, req *Request, resp *Response, _ Advice) error {
			return r.runLoop(ctx, req, resp)
		}
		if err := r.AroundAdvice(r.Vars, req, resp, next); err != nil {
			return err
		}
	} else {
		if err := r.runLoop(ctx, req, resp); err != nil {
			return err
		}
	}
	if r.AfterAdvice != nil {
		if err := r.AfterAdvice(r.Vars, req, resp, noop); err != nil {
			return err
		}
	}

	return nil
}

func (r *Agent) runLoop(ctx context.Context, req *Request, resp *Response) error {
	var history []*Message

	// system role prompt as first message
	if r.Instruction != "" {
		role := r.Role
		if role == "" {
			role = RoleSystem
		}
		history = append(history, &Message{
			Role:    role,
			Content: r.Instruction,
			Sender:  r.Name,
		})
	}
	history = append(history, r.sw.History...)

	if req.Message == nil {
		req.Message = &Message{
			Role:    RoleUser,
			Content: req.RawInput.Input(),
			Sender:  r.Name,
		}
	}
	history = append(history, req.Message)

	initLen := len(history)
	agentRole := r.Role
	if agentRole == "" {
		agentRole = RoleAssistant
	}

	runTool := func(ctx context.Context, name string, args map[string]any) (*Result, error) {
		log.Debugf("run tool: %s %+v\n", name, args)
		return r.runTool(ctx, name, args)
	}

	result, err := llm.Call(ctx, &llm.Request{
		BaseUrl:  r.Model.BaseUrl,
		ApiKey:   r.Model.ApiKey,
		Model:    r.Model.Name,
		Messages: history,
		MaxTurns: r.MaxTurns,
		RunTool:  runTool,
		Tools:    r.Functions,
	})
	if err != nil {
		return err
	}

	if !result.Transfer {
		message := Message{
			Role:    result.Role,
			Content: result.Content,
			Sender:  r.Name,
		}
		history = append(history, &message)
	}

	resp.Messages = history[initLen:]

	resp.Agent = r
	resp.Transfer = result.Transfer
	resp.NextAgent = result.NextAgent

	r.sw.History = history
	return nil
}

func (r *Agent) runTool(ctx context.Context, name string, args map[string]any) (*Result, error) {
	var out string
	var err error

	if fn, ok := r.Vars.FuncRegistry[name]; ok {
		log.Debugf("run function: %s %+v\n", name, args)
		return fn(ctx, r, name, args)
	}

	switch name {
	case transferAgentName:
		return transferAgent(ctx, r, name, args)
	default:
		out, err = runCommandTool(ctx, r, name, args)
	}

	if err != nil {
		return &Result{
			Value: fmt.Sprintf("%s: %v", out, err),
		}, nil
	}
	return &api.Result{
		Value: out,
	}, nil
}
