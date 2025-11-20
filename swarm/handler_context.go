package swarm

import (
	"time"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func ContextMiddleware(sw *Swarm) api.Middleware {

	return func(agent *api.Agent, next Handler) Handler {
		maxHistory := agent.Arguments.GetInt("max_history")
		maxSpan := agent.Arguments.GetInt("max_span")
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())
			logger.Debugf("🔗 (context): %s max_history: %v max_span: %v\n", agent.Name, maxHistory, maxSpan)

			var id = sw.ID
			var env = sw.globalEnv()

			var history []*api.Message

			// 1. New System Message
			// system role prompt as first message
			var prompt *api.Message
			if agent.Instruction != nil {
				prompt = &api.Message{
					ID:      uuid.NewString(),
					Session: id,
					Created: time.Now(),
					//
					Role:    api.RoleSystem,
					Content: agent.Instruction.Content,
					Sender:  agent.Name,
				}
				history = append(history, prompt)
			}

			// 2. Context Messages
			// skip system role
			for i, msg := range sw.Vars.ListHistory() {
				if msg.Role != api.RoleSystem {
					logger.Debugf("adding [%v]: %s %s (%v)\n", i, msg.Role, abbreviate(msg.Content, 100), len(msg.Content))
					history = append(history, msg)
				}
			}

			// 3. New User Message
			// Additional user message
			var message = &api.Message{
				ID:      uuid.NewString(),
				Session: id,
				Created: time.Now(),
				//
				Role:    api.RoleUser,
				Content: req.Query(),
				Sender:  sw.User.Email,
			}
			history = append(history, message)

			logger.Infof("• context messages: %v\n", len(history))
			if logger.IsTrace() {
				for i, v := range history {
					logger.Debugf("[%v] %+v\n", i, v)
				}
			}

			// request
			nreq := req.Clone()
			nreq.Name = agent.Name
			// nreq.MaxTurns = agent.MaxTurns
			nreq.Tools = agent.Tools
			nreq.Runner = agent.Runner
			nreq.Arguments.Add(env)
			nreq.Vars = sw.Vars

			//
			initLen := len(history)
			nreq.Messages = history

			// call next
			if err := next.Serve(nreq, resp); err != nil {
				return err
			}

			if resp.Result == nil {
				resp.Result = &api.Result{}
			}
			var result = resp.Result

			// Response
			if result.State != api.StateTransfer {
				message := api.Message{
					ID:      uuid.NewString(),
					Session: id,
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

			sw.Vars.AddHistory(history)
			//
			resp.Messages = history[initLen:]
			resp.Agent = agent
			resp.Result = result

			return nil
		})
	}
}
