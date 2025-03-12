package swarm

import (
	"context"
	"fmt"
	"strings"

	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
)

const transferAgentName = "agent_transfer"
const (
	ToolLabelSystem = "system"
	ToolLabelMcp    = "mcp"
	ToolLabelAgent  = "agent"
	ToolLabelFunc   = "func"
)

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

	RawInput *UserInput

	// Vars *Vars

	// Functions that the agent can call
	Tools map[string]*ToolFunc

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

func (r *Agent) Vars() *Vars {
	return r.sw.Vars
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
		if err := r.BeforeAdvice(r.sw.Vars, req, resp, noop); err != nil {
			return err
		}
	}
	if r.AroundAdvice != nil {
		next := func(vars *Vars, req *Request, resp *Response, _ Advice) error {
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

func (r *Agent) runLoop(ctx context.Context, req *Request, resp *Response) error {
	// "resource:" prefix is used to refer to a resource
	// "vars:" prefix is used to refer to a variable
	apply := func(s string, vars *Vars) (string, error) {
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

	var history []*Message

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
		history = append(history, &Message{
			Role:    role,
			Content: content,
			Sender:  r.Name,
		})
	}
	// FIXME: this is confusing LLM?
	// history = append(history, r.sw.History...)

	if req.Message == nil {
		req.Message = &Message{
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

	runTool := func(ctx context.Context, name string, args map[string]any) (*Result, error) {
		log.Debugf("run tool: %s %+v\n", name, args)
		return r.callTool(ctx, name, args)
	}

	result, err := llm.Send(ctx, &api.Request{
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
		message := Message{
			ContentType: result.ContentType,
			Role:        result.Role,
			Content:     result.Content,
			Sender:      r.Name,
		}
		history = append(history, &message)
	}

	resp.Messages = history[initLen:]

	resp.Agent = r
	resp.Result = result.Result

	r.sw.History = history
	return nil
}

func (r *Agent) callTool(ctx context.Context, toolName string, args map[string]any) (*Result, error) {

	// if toolName == transferAgentName {
	// 	return transferAgentSub(ctx, r, args)
	// }

	v, ok := r.Tools[toolName]
	if !ok {
		return nil, fmt.Errorf("no such tool: %s", toolName)
	}

	switch v.Label {
	case ToolLabelAgent:
		return &api.Result{
			NextAgent: v.Service,
			State:     api.StateTransfer,
		}, nil
	case ToolLabelMcp:
		out, err := r.sw.McpServerTool.CallTool(v.Service, v.Func, args)
		return &api.Result{
			Value: out,
		}, err
	case ToolLabelSystem:
		out, err := CallSystemTool(r.sw.fs, ctx, r, v.Func, args)
		return &api.Result{
			Value: out,
		}, err
	case ToolLabelFunc:
		funcName := v.Func
		if fn, ok := r.sw.FuncRegistry[funcName]; ok {
			return fn(ctx, r, funcName, args)
		}
	}

	return nil, fmt.Errorf("no such tool: %s", toolName)
}

// func transferAgentSub(ctx context.Context, agent *Agent, props map[string]any) (*Result, error) {
// 	transferTo, err := GetStrProp("agent", props)

// 	if err != nil {
// 		return nil, err
// 	}
// 	return &api.Result{
// 		NextAgent: fmt.Sprintf("%s/%s", agent.Name, transferTo),
// 		State:     api.StateTransfer,
// 	}, nil
// }
