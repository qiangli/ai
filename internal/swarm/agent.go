package swarm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
)

const TransferAgentName = "agent_transfer"

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
}

func (r *Agent) Serve(req *Request, resp *Response) error {
	log.Debugf("run agent: %s\n", r.Name)

	ctx := req.Context()

	if r.Entrypoint != nil {
		if err := r.Entrypoint(r.Vars, r, req.RawInput); err != nil {
			return err
		}
	}

	// dependencies
	if len(r.Dependencies) > 0 {
		for _, dep := range r.Dependencies {
			depReq := &Request{
				Agent:    dep.Name,
				RawInput: req.RawInput,
				Message:  req.Message,
			}
			depResp := &Response{}
			if err := dep.Serve(depReq, depResp); err != nil {
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
	var history = []*Message{}

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
	if req.Message == nil {
		req.Message = &Message{
			Role:    RoleUser,
			Content: req.RawInput.Input(),
		}
	}
	history = append(history, req.Message)

	initLen := len(history)
	agentRole := r.Role
	if agentRole == "" {
		agentRole = RoleAssistant
	}

	runTool := func(ctx context.Context, name string, args map[string]any) (string, error) {
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

	message := Message{
		Role:    result.Role,
		Content: result.Content,
		Sender:  r.Name,
	}
	history = append(history, &message)

	if resp == nil {
		return nil
	}

	resp.Messages = history[initLen:]
	resp.Agent = r
	resp.Transfer = result.Transfer
	resp.NextAgent = result.NextAgent
	return nil
}

func (r *Agent) runTool(ctx context.Context, name string, args map[string]any) (string, error) {
	var out string
	var err error

	switch name {
	case TransferAgentName:
		out, err = transferAgent(ctx, r, name, args)
	default:
		out, err = runCommandTool(ctx, r, name, args)
	}

	if err != nil {
		return fmt.Sprintf("%s: %v", out, err), nil
	}
	return out, nil
}
