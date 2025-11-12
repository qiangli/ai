package swarm

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func ContextMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {

	return func(agent *api.Agent) api.Middleware {

		mustResolveContext := func(parent *api.Agent, req *api.Request, s string) ([]*api.Message, error) {
			name, query, found := parseAgentCommand(s)
			if !found {
				return nil, fmt.Errorf("invalid context: %s", s)
			}
			out, err := sw.callAgent(parent, req, name, query)
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

				env := sw.globalEnv()

				var chatID = sw.ChatID
				var history []*api.Message

				// 1. New System Message
				// system role prompt as first message
				if agent.Instruction != nil {
					history = append(history, &api.Message{
						ID:      uuid.NewString(),
						ChatID:  chatID,
						Created: time.Now(),
						//
						Role:    api.RoleSystem,
						Content: agent.Instruction.Content,
						Sender:  agent.Name,
					})
				}

				// 2. Historical Messages
				// support dynamic context history
				// skip system role
				if !agent.New() {
					var list []*api.Message
					var emoji = "•"
					if agent.Context != "" {
						// continue without context if failed
						if resolved, err := mustResolveContext(agent, req, agent.Context); err != nil {
							logger.Errorf("failed to resolve context %s: %v\n", agent.Context, err)
						} else {
							list = resolved
							emoji = "🤖"
						}
					} else {
						list = sw.Vars.History
					}
					if len(list) > 0 {
						logger.Debugf("%s context messages: %v\n", emoji, len(list))
						for i, msg := range list {
							if msg.Role != api.RoleSystem {
								logger.Debugf("adding [%v]: %s %s (%v)\n", i, msg.Role, abbreviate(msg.Content, 100), len(msg.Content))
								history = append(history, msg)
							}
						}
					}
				}

				// 3. New User Message
				// Additional user message
				history = append(history, &api.Message{
					ID:      uuid.NewString(),
					ChatID:  chatID,
					Created: time.Now(),
					//
					Role:    api.RoleUser,
					Content: req.Query,
					Sender:  agent.Name,
				})

				logger.Debugf("messages total: %v\n", len(history))

				if logger.IsTrace() {
					for i, v := range history {
						logger.Debugf("[%v] %+v\n", i, v)
					}
				}

				// Request
				initLen := len(history)

				//
				var runTool = sw.createCaller(sw.User, agent)

				nreq := req.Clone()
				nreq.Name = agent.Name
				nreq.Messages = history
				nreq.MaxTurns = agent.MaxTurns
				nreq.Tools = agent.Tools
				nreq.RunTool = runTool
				nreq.Arguments = env
				nreq.Vars = sw.Vars

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

				sw.Vars.History = history
				//
				resp.Messages = history[initLen:]
				resp.Agent = agent
				resp.Result = result

				return nil
			})
		}
	}
}
