package swarm

import (
	"context"
	"maps"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/log"
)

func (h *agentHandler) action(ctx context.Context, req *api.Request, resp *api.Response) error {
	var r = h.agent

	// apply template/load
	// apply := func(vars *api.Vars, ext, s string) (string, error) {
	// 	//
	// 	if ext == "tpl" {
	// 		// TODO custom template func?
	// 		return applyTemplate(s, vars, tplFuncMap)
	// 	}
	// 	return s, nil
	// }

	// var chatID = h.vars.ChatID
	// var history []*api.Message

	// // 1. New System Message
	// // system role prompt as first message
	// if r.Instruction != nil {
	// 	// update the request instruction
	// 	content, err := apply(h.vars, r.Instruction.Type, r.Instruction.Content)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	// dynamic @prompt if requested
	// 	content, err = h.resolvePrompt(ctx, req, content)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	history = append(history, &api.Message{
	// 		ID:      uuid.NewString(),
	// 		ChatID:  chatID,
	// 		Created: time.Now(),
	// 		//
	// 		Role:    api.RoleSystem,
	// 		Content: content,
	// 		Sender:  r.Name,
	// 	})
	// 	log.GetLogger(ctx).Debugf("Added system role message: %v\n", len(history))
	// }

	// // 2. Historical Messages
	// // support dynamic context history
	// // skip system role
	// if !r.New {
	// 	var list []*api.Message
	// 	var emoji = "•"
	// 	if r.Context != "" {
	// 		// continue without context if failed
	// 		if resolved, err := h.mustResolveContext(ctx, req, r.Context); err != nil {
	// 			log.GetLogger(ctx).Errorf("failed to resolve context %s: %v\n", r.Context, err)
	// 		} else {
	// 			list = resolved
	// 			emoji = "🤖"
	// 		}
	// 	} else {
	// 		list = h.vars.History
	// 	}
	// 	if len(list) > 0 {
	// 		log.GetLogger(ctx).Infof("%s context messages: %v\n", emoji, len(list))
	// 		for i, msg := range list {
	// 			if msg.Role != api.RoleSystem {
	// 				history = append(history, msg)
	// 				log.GetLogger(ctx).Debugf("Added historical message: %v %s %s\n", i, msg.Role, head(msg.Content, 100))
	// 			}
	// 		}
	// 	}
	// }

	// // 3. New User Message
	// // Additional user message
	// if r.Message != "" {
	// 	req.Messages = append(req.Messages, &api.Message{
	// 		ID:      uuid.NewString(),
	// 		ChatID:  chatID,
	// 		Created: time.Now(),
	// 		//
	// 		Role:    api.RoleUser,
	// 		Content: r.Message,
	// 		Sender:  r.Name,
	// 	})
	// }

	// req.Messages = append(req.Messages, &api.Message{
	// 	ID:      uuid.NewString(),
	// 	ChatID:  chatID,
	// 	Created: time.Now(),
	// 	//
	// 	Role:    api.RoleUser,
	// 	Content: req.RawInput.Query(),
	// 	Sender:  r.Name,
	// })

	// merge args
	var args map[string]any
	if r.Arguments != nil || req.Arguments != nil {
		args = make(map[string]any)
		maps.Copy(args, r.Arguments)
		maps.Copy(args, req.Arguments)
	}
	// check agents in args
	for key, val := range args {
		if v, ok := val.(string); ok {
			resolved, err := h.resolveArgument(ctx, req, v)
			if err != nil {
				return err
			}
			args[key] = resolved
		}
	}

	// history = append(history, req.Messages...)
	// log.GetLogger(ctx).Debugf("Added user role message: %v\n", len(history))

	// Request
	// initLen := len(history)

	//
	// var runTool = h.createCaller()

	// // resolve if model is @agent
	// var model *api.Model
	// if v, err := h.resolveModel(ctx, req, r.Model); err != nil {
	// 	return err
	// } else {
	// 	model = v
	// }
	// var request = llm.Request{
	// 	Agent:    r.Name,
	// 	Messages: history,
	// 	MaxTurns: r.MaxTurns,
	// 	Tools:    r.Tools,
	// 	//
	// 	Model: model,
	// 	//
	// 	RunTool: runTool,
	// 	// agent tool
	// 	Arguments: args,
	// 	//
	// 	Vars: h.vars,
	// }

	// // openai/tts
	// if r.Instruction != nil {
	// 	request.Instruction = r.Instruction.Content
	// }
	// request.Query = r.RawInput.Query()

	// var adapter llm.LLMAdapter = adapter.Chat
	// if h.agent.Adapter != "" {
	// 	if v, err := h.sw.Adapters.Get(h.agent.Adapter); err == nil {
	// 		adapter = v
	// 	} else {
	// 		return err
	// 	}
	// }

	// // LLM adapter
	// // TODO model <-> adapter
	// result, err := adapter(ctx, &request)
	// if err != nil {
	// 	return err
	// }
	// if result.Result == nil {
	// 	return fmt.Errorf("Empty response")
	// }

	// // Response
	// if result.Result.State != api.StateTransfer {
	// 	message := api.Message{
	// 		ID:      uuid.NewString(),
	// 		ChatID:  chatID,
	// 		Created: time.Now(),
	// 		//
	// 		ContentType: result.Result.MimeType,
	// 		Content:     result.Result.Value,
	// 		// TODO encode result.Result.Content
	// 		Role:   nvl(result.Role, api.RoleAssistant),
	// 		Sender: r.Name,
	// 	}
	// 	// TODO add Value field to message?
	// 	history = append(history, &message)
	// }
	//
	// h.vars.Extra[extraResult] = result.Result.Value
	// h.vars.History = history

	// //
	// resp.Messages = history[initLen:]
	// resp.Agent = r
	// resp.Result = result.Result
	return nil
}

func (h *agentHandler) flowSequence(req *api.Request, resp *api.Response) error {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	// var shared = make(map[string]any)

	for _, action := range h.agent.Flow.Actions {
		if action.Tool.Type == api.ToolTypeAgent {
			req := api.NewRequest(ctx, action.Tool.Name, h.agent.RawInput.Clone())
			req.Parent = h.agent

			resp := &api.Response{}
			if err := h.exec(req, resp); err != nil {
				return err
			}
		}

		// args := make(map[string]any)
		// h.callTool(ctx, action.Tool, args)
	}
	return nil
}

func (h *agentHandler) flowParallel(req *api.Request, resp *api.Response) error {
	return nil
}

func (h *agentHandler) flowChoice(req *api.Request, resp *api.Response) error {
	return nil
}

func (h *agentHandler) flowMap(req *api.Request, resp *api.Response) error {
	return nil
}
