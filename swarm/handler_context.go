package swarm

import (
	"encoding/json"
	"fmt"
	"maps"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func ContextMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {

	return func(agent *api.Agent) api.Middleware {

		mustResolveContext := func(parent *api.Agent, req *api.Request, s string) ([]*api.Message, error) {
			at, found := parseAgentCommand(s)
			if !found {
				return nil, fmt.Errorf("invalid context: %s", s)
			}
			nreq := req.Clone()
			if len(at.Arguments) > 0 {
				if nreq.Arguments == nil {
					at.Arguments = make(map[string]any)
				}
				maps.Copy(nreq.Arguments, at.Arguments)
			}
			out, err := sw.callAgent(parent, req, at.Name, at.Message)
			if err != nil {
				return nil, err
			}
			var list []*api.Message
			if err := json.Unmarshal([]byte(out), &list); err != nil {
				return nil, err
			}
			return list, nil
		}

		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				logger := log.GetLogger(req.Context())
				logger.Debugf("🔗 (context): %s max_history: %v max_span: %v\n", agent.Name, agent.MaxHistory, agent.MaxSpan)

				var chatID = sw.ChatID
				var env = sw.globalEnv()

				var history []*api.Message

				// 1. New System Message
				// system role prompt as first message
				var prompt *api.Message
				if agent.Instruction != nil {
					prompt = &api.Message{
						ID:      uuid.NewString(),
						ChatID:  chatID,
						Created: time.Now(),
						//
						Role:    api.RoleSystem,
						Content: agent.Instruction.Content,
						Sender:  agent.Name,
					}
					history = append(history, prompt)
				}

				// 2. Historical Messages
				// skip system role
				for i, msg := range sw.Vars.History {
					if msg.Role != api.RoleSystem {
						logger.Debugf("adding [%v]: %s %s (%v)\n", i, msg.Role, abbreviate(msg.Content, 100), len(msg.Content))
						history = append(history, msg)
					}
				}

				// if !agent.New() {
				// 	var list []*api.Message
				// 	var emoji = "•"
				// 	if agent.Context != "" {
				// 		// continue without context if failed
				// 		if resolved, err := mustResolveContext(agent, req, agent.Context); err != nil {
				// 			logger.Errorf("failed to resolve context %s: %v\n", agent.Context, err)
				// 		} else {
				// 			list = resolved
				// 			emoji = "🤖"
				// 		}
				// 	} else {
				// 		list = sw.Vars.History
				// 	}
				// 	if len(list) > 0 {
				// 		logger.Debugf("%s context messages: %v\n", emoji, len(list))
				// 		for i, msg := range list {
				// 			if msg.Role != api.RoleSystem {
				// 				logger.Debugf("adding [%v]: %s %s (%v)\n", i, msg.Role, abbreviate(msg.Content, 100), len(msg.Content))
				// 				history = append(history, msg)
				// 			}
				// 		}
				// 	}
				// }

				// 3. New User Message
				// Additional user message
				var message = &api.Message{
					ID:      uuid.NewString(),
					ChatID:  chatID,
					Created: time.Now(),
					//
					Role:    api.RoleUser,
					Content: req.Query,
					Sender:  agent.Name,
				}
				history = append(history, message)

				var emoji = "•"
				// override if context agent is specified
				if agent.Context != "" {
					if resolved, err := mustResolveContext(agent, req, agent.Context); err != nil {
						logger.Errorf("failed to resolve context %s: %v\n", agent.Context, err)
					} else {
						history = resolved
						emoji = "🤖"
					}
				}

				logger.Infof("%s context messages: %v\n", emoji, len(history))
				if logger.IsTrace() {
					for i, v := range history {
						logger.Debugf("[%v] %+v\n", i, v)
					}
				}

				// request
				nreq := req.Clone()
				nreq.Name = agent.Name
				nreq.MaxTurns = agent.MaxTurns
				nreq.Tools = agent.Tools
				nreq.RunTool = sw.createCaller(sw.User, agent)
				nreq.Arguments = env
				nreq.Vars = sw.Vars

				//
				initLen := len(history)
				nreq.Messages = history

				// call next
				if err := next.Serve(nreq, resp); err != nil {
					return err
				}

				if resp.Result == nil {
					return fmt.Errorf("Empty response")
				}

				var result = resp.Result

				// Response
				if result.State != api.StateTransfer {
					message := api.Message{
						ID:      uuid.NewString(),
						ChatID:  chatID,
						Created: time.Now(),
						//
						ContentType: result.MimeType,
						Content:     result.Value,
						// TODO encode result.Result.Content
						Role:   nvl(result.Role, api.RoleAssistant),
						Sender: agent.Name,
					}
					// TODO add Value field to message?
					history = append(history, &message)
				}

				// always append: mem will save the diff
				sw.Vars.History = append(sw.Vars.History, history...)
				//
				resp.Messages = history[initLen:]
				resp.Agent = agent
				resp.Result = result

				return nil
			})
		}
	}
}
