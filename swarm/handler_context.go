package swarm

import (
	// "bytes"
	// "context"
	// "encoding/json"
	"fmt"
	"strings"
	// "text/template"
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/llm"
	// "github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
	// "fmt"
	// "github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/log"
)

// ContextMiddlewareFunc loads the dynamical modle
func ContextMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {
	return func(agent *api.Agent) api.Middleware {
		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				ctx := req.Context()
				log.GetLogger(ctx).Debugf("🟦 (context): %s adapter: %s\n", agent.Name)

				// var ctx = req.Context()
				// var r = h.agent

				env := sw.globalEnv()
				// h.mapAssign(req, env, req.Arguments, false)

				// apply template/load
				// TODO  vars -> data may break some existing config
				applyGlobal := func(ext, s string) (string, error) {
					if strings.HasPrefix(s, "#!") {
						parts := strings.SplitN(s, "\n", 2)
						if len(parts) == 2 {
							// remove hashbang line
							// return h.applyGlobal(parts[1])
							return applyTemplate(sw.template, parts[1], env)
						}
						// remove hashbang
						return applyTemplate(sw.template, parts[0][2:], env)
					}
					if ext == "tpl" {
						return applyTemplate(sw.template, s, env)
					}
					return s, nil
				}

				resolveGlobal := func(ext, s string) (string, error) {
					// update the request instruction
					content, err := applyGlobal(ext, s)
					if err != nil {
						return "", err
					}

					// dynamic @prompt if requested
					content, err = sw.resolvePrompt(agent, req, content)
					if err != nil {
						return "", err
					}
					return content, nil
				}

				var chatID = sw.ChatID
				var history []*api.Message
				var instructions []string

				// 1. New System Message
				// system role prompt as first message
				// inherit embedded agent instructions
				addContext := func(in *api.Instruction, sender string) error {
					content, err := resolveGlobal(in.Type, in.Content)
					if err != nil {
						return err
					}

					// update instruction
					instructions = append(instructions, content)

					history = append(history, &api.Message{
						ID:      uuid.NewString(),
						ChatID:  chatID,
						Created: time.Now(),
						//
						Role:    api.RoleSystem,
						Content: content,
						Sender:  sender,
					})
					log.GetLogger(ctx).Debugf("Added system role message: %v\n", len(history))
					return nil
				}
				for _, v := range agent.Embed {
					if v.Instruction != nil {
						addContext(v.Instruction, agent.Name)
					}
				}
				if agent.Instruction != nil {
					addContext(agent.Instruction, agent.Name)
				}

				// 2. Historical Messages
				// support dynamic context history
				// skip system role
				if !agent.New {
					var list []*api.Message
					var emoji = "•"
					if agent.Context != "" {
						// continue without context if failed
						if resolved, err := sw.mustResolveContext(agent, req, agent.Context); err != nil {
							log.GetLogger(ctx).Errorf("failed to resolve context %s: %v\n", agent.Context, err)
						} else {
							list = resolved
							emoji = "🤖"
						}
					} else {
						list = sw.Vars.History
					}
					if len(list) > 0 {
						log.GetLogger(ctx).Infof("%s context messages: %v\n", emoji, len(list))
						for i, msg := range list {
							if msg.Role != api.RoleSystem {
								history = append(history, msg)
								log.GetLogger(ctx).Debugf("Added historical message [%v]: %s %s\n", i, msg.Role, head(msg.Content, 100))
							}
						}
					}
				}

				// 3. New User Message
				// Additional user message
				// embeded messages not inherited for now
				if agent.Message != "" {
					msg, err := resolveGlobal("", agent.Message)
					if err != nil {
						return err
					}
					history = append(history, &api.Message{
						ID:      uuid.NewString(),
						ChatID:  chatID,
						Created: time.Now(),
						//
						Role:    api.RoleUser,
						Content: msg,
						Sender:  agent.Name,
					})
				}

				history = append(history, &api.Message{
					ID:      uuid.NewString(),
					ChatID:  chatID,
					Created: time.Now(),
					//
					Role:    api.RoleUser,
					Content: req.RawInput.Query(),
					Sender:  agent.Name,
				})

				log.GetLogger(ctx).Debugf("Added messages: %v\n", len(history))

				// Request
				initLen := len(history)

				//
				var runTool = sw.createCaller(sw.User, agent)

				// // resolve if model is @agent
				// var model *api.Model
				// if v, err := h.resolveModel(ctx, req, agent.Model); err != nil {
				// 	return err
				// } else {
				// 	model = v
				// }

				// model := h.agent.Model

				// ak, err := h.sw.Secrets.Get(h.agent.Owner, model.ApiKey)
				// if err != nil {
				// 	return err
				// }
				// token := func() string {
				// 	return ak
				// }

				// var request = llm.Request{
				// 	Name:     agent.Name,
				// 	Messages: history,
				// 	MaxTurns: agent.MaxTurns,
				// 	Tools:    agent.Tools,
				// 	//
				// 	// Model: model,
				// 	// Token: token,
				// 	//
				// 	RunTool: runTool,
				// 	// agent tool
				// 	Arguments: env,
				// 	//
				// 	Vars: h.sw.Vars,
				// }

				nreq := req.Clone()
				nreq.Name = agent.Name
				nreq.Messages = history
				nreq.MaxTurns = agent.MaxTurns
				nreq.Tools = agent.Tools
				nreq.RunTool = runTool
				nreq.Arguments = env
				nreq.Vars = sw.Vars

				// openai/tts
				if len(instructions) > 0 {
					nreq.Instruction = strings.Join(instructions, "\n")
				}
				nreq.Query = agent.RawInput.Query()

				//
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
				// result, err := adapter(ctx, nreq)

				// // client
				// if err != nil {
				// 	return err
				// }

				if err := next.Serve(req, resp); err != nil {
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
