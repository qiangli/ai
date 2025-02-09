package swarm

import (
	"context"
	// "encoding/json"
	// "fmt"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/llm"
)

type Swarm struct {
	Agent    *Agent
	Messages []*Message
	Vars     *Vars
	MaxTurns int
	MaxTime  int64
	Stream   bool
	Debug    bool
}

func NewSwarm(agent *Agent, vars *Vars, messages []*Message) *Swarm {
	return &Swarm{
		Agent:    agent,
		Vars:     vars,
		Messages: messages,
		MaxTurns: 100,
		MaxTime:  3600,
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

	messages := []*Message{}
	if r.Agent.Instruction != "" {
		role := r.Agent.Role
		if role == "" {
			role = RoleSystem
		}
		messages = append(messages, &Message{
			Role:    role,
			Content: r.Agent.Instruction,
			Sender:  r.Agent.Name,
		})
	}
	if req.Message != nil {
		messages = append(messages, req.Message)
	}

	initLen := len(r.Messages)
	activeAgent := r.Agent
	agentRole := activeAgent.Role
	if agentRole == "" {
		agentRole = RoleAssistant
	}

	runTool := func(ctx context.Context, name string, args map[string]interface{}) (string, error) {
		content, err := llm.RunTool(&internal.ToolConfig{
			Model: &internal.Model{
				Name:    activeAgent.Model.Name,
				BaseUrl: activeAgent.Model.BaseUrl,
				ApiKey:  activeAgent.Model.ApiKey,
			},
		}, ctx, name, args)
		if err != nil {
			return "", err
		}
		return content, nil
	}

	result, err := llm.Call(ctx, &llm.Request{
		BaseUrl:  activeAgent.Model.BaseUrl,
		ApiKey:   activeAgent.Model.ApiKey,
		Model:    activeAgent.Model.Name,
		History:  history,
		Messages: messages,
		MaxTurns: r.MaxTurns,
		RunTool:  runTool,
		Tools:    activeAgent.Functions,
	})
	if err != nil {
		return err
	}

	history = append(history, messages...)
	message := Message{
		Role:    result.Role,
		Content: result.Content,
		Sender:  activeAgent.Name,
	}
	history = append(history, &message)

	// for len(history)-initLen < r.MaxTurns {

	// 	resp, err := llm.Call(ctx, &llm.Request{
	// 		BaseUrl:  activeAgent.Model.BaseUrl,
	// 		ApiKey:   activeAgent.Model.ApiKey,
	// 		Model:    activeAgent.Model.Name,
	// 		History:  history,
	// 		Messages: messages,
	// 		Tools:    activeAgent.Functions,
	// 	})

	// 	if err != nil {
	// 		return err
	// 	}

	// 	history = append(history, messages...)

	// 	if len(resp.ToolCalls) == 0 {
	// 		break
	// 	}

	// 	message := Message{
	// 		Role:    resp.Role,
	// 		Content: resp.Content,
	// 		Sender:  activeAgent.Name,
	// 		ToolCalls: resp.ToolCalls,
	// 	}
	// 	history = append(history, &message)

	// 	// handle function calls, updating context_variables
	// 	messages, err = r.runTools(ctx, activeAgent, resp.ToolCalls, req.Vars)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	if resp == nil {
		return nil
	}

	resp.Messages = history[initLen:]
	resp.Agent = activeAgent
	resp.Vars = req.Vars
	return nil
}

// func (r *Swarm) runTools(ctx context.Context, agent *Agent, calls []*ToolCall, _ *Vars) ([]*Message, error) {
// 	functions := agent.Functions
// 	var funcMap = make(map[string]bool)
// 	for _, v := range functions {
// 		funcMap[v.Name] = true
// 	}

// 	var messages []*Message
// 	for _, v := range calls {
// 		var args map[string]interface{}
// 		if err := json.Unmarshal([]byte(v.Function.Arguments), &args); err != nil {
// 			return nil, err
// 		}
// 		var name = v.Function.Name

// 		if _, ok := funcMap[name]; !ok {
// 			messages = append(messages, &Message{
// 				Role:     RoleTool,
// 				Content:  fmt.Sprintf("error: tool %s not found", name),
// 				ToolCall: v,
// 			})
// 			continue
// 		}

// 		// content, err := RunTool(ctx, agent, name, args)
// 		content, err := llm.RunTool(&internal.ToolConfig{
// 			Model: &internal.Model{
// 				Name:    agent.Model.Name,
// 				BaseUrl: agent.Model.BaseUrl,
// 				ApiKey:  agent.Model.ApiKey,
// 			},
// 		}, ctx, name, args)

// 		if err != nil {
// 			return nil, err
// 		}
// 		messages = append(messages, &Message{
// 			Role:     RoleTool,
// 			Content:  content,
// 			ToolCall: v,
// 		})
// 	}
// 	return messages, nil
// }
