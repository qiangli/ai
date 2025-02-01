package swarm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/internal/llm"
)

type Swarm struct {
	Agent    *Agent
	Messages []*Message
	Vars     *Vars
	MaxTurns int
	Stream   bool
	Debug    bool
}

func NewSwarm(agent *Agent, vars *Vars, messages []*Message) *Swarm {
	return &Swarm{
		Agent:    agent,
		Vars:     vars,
		Messages: messages,
		MaxTurns: 100,
		Stream:   true,
		Debug:    false,
	}
}

func (r *Swarm) Run(req *Request) (*Response, error) {
	ctx := context.TODO()

	if r.Agent.BeforeAdvice != nil {
		if err := r.Agent.BeforeAdvice(req, nil, nil); err != nil {
			return nil, err
		}
	}

	var resp Response

	if r.Agent.AroundAdvice != nil {
		next := func(_ *Request, _ *Response, _ Advice) error {
			return r.runLoop(ctx, req, &resp)
		}
		if err := r.Agent.AroundAdvice(req, &resp, next); err != nil {
			return nil, err
		}
	} else {
		if err := r.runLoop(ctx, req, &resp); err != nil {
			return nil, err
		}
	}

	if r.Agent.AfterAdvice != nil {
		if err := r.Agent.AfterAdvice(nil, &resp, nil); err != nil {
			return nil, err
		}
	}

	return &resp, nil
}

func (r *Swarm) runLoop(ctx context.Context, req *Request, resp *Response) error {
	var history = []*Message{}
	history = append(history, r.Messages...)

	initLen := len(r.Messages)
	activeAgent := r.Agent

	for len(history)-initLen < r.MaxTurns {
		instruction := []*Message{
			{Role: RoleSystem,
				Content: r.Agent.Instruction,
				Sender:  r.Agent.Name,
			},
		}
		history = append(history, req.Message)
		resp, err := llm.Call(ctx, &llm.Request{
			BaseUrl:  activeAgent.Model.BaseUrl,
			ApiKey:   activeAgent.Model.ApiKey,
			Model:    activeAgent.Model.Name,
			Messages: append(instruction, history...),
			Tools:    activeAgent.Functions,
		})

		if err != nil {
			return err
		}

		message := Message{
			Role:    activeAgent.Role,
			Content: resp.Content,
			Sender:  activeAgent.Name,
		}

		history = append(history, &message)

		if len(resp.ToolCalls) == 0 {
			break
		}

		// handle function calls, updating context_variables
		partial, err := runTools(ctx, resp.ToolCalls, activeAgent.Functions, req.Vars)
		if err != nil {
			return err
		}
		history = append(history, partial...)
	}

	if resp == nil {
		return nil
	}

	resp.Messages = history[initLen:]
	resp.Agent = activeAgent
	resp.Vars = req.Vars
	return nil
}

func runTools(ctx context.Context, calls []*ToolCall, functions []*ToolFunc, _ *Vars) ([]*Message, error) {
	var funcMap = make(map[string]bool)
	for _, v := range functions {
		funcMap[v.Name] = true
	}

	var messages []*Message
	for _, v := range calls {
		var name = v.Name
		var args = v.Arguments

		if _, ok := funcMap[name]; ok {
			messages = append(messages, &Message{
				Role:     RoleTool,
				Content:  fmt.Sprintf("error: tool %s not found", name),
				ToolCall: v,
			})
			continue
		}

		content, err := runTool(ctx, name, args)
		if err != nil {
			return nil, err
		}
		messages = append(messages, &Message{
			Role:     RoleTool,
			Content:  content,
			ToolCall: v,
		})
	}
	return messages, nil
}

func runTool(ctx context.Context, name string, args map[string]interface{}) (string, error) {
	return llm.RunTool(nil, ctx, name, args)
}
