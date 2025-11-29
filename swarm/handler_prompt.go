package swarm

import (
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

// System role prompt
func InstructionMiddleware(sw *Swarm) api.Middleware {
	resolve := func(parent *api.Agent, req *api.Request, s string) (string, error) {
		if !conf.IsAgentTool(s) {
			return s, nil
		}
		out, err := sw.expandx(req.Context(), parent, s)
		if err != nil {
			return "", err
		}
		return out, nil
	}
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())
			logger.Debugf("🔗 (instruction): %s\n", agent.Name)

			var env = req.Arguments.GetAllArgs()
			var instructions []string

			add := func(in string) error {
				content, err := atm.ApplyTemplate(agent.Template, in, env)
				if err != nil {
					return err
				}

				// dynamic @ if requested
				content, err = resolve(agent, req, content)
				if err != nil {
					return err
				}

				// update instruction
				instructions = append(instructions, content)
				return nil
			}

			var addAll func(*api.Agent) error

			// inherit embedded agent instructions
			// merge all including the current agent
			addAll = func(a *api.Agent) error {
				for _, v := range a.Embed {
					if err := addAll(v); err != nil {
						return err
					}
				}
				in := a.Instruction
				if in != "" {
					if err := add(in); err != nil {
						return err
					}
				}
				return nil
			}

			var prompt = agent.Prompt()
			if prompt == "" {
				if err := addAll(agent); err != nil {
					return err
				}

				prompt = strings.Join(instructions, "\n")
			}
			agent.SetPrompt(prompt)

			logger.Debugf("instructions (%v): %s (%v)\n", len(instructions), abbreviate(prompt, 64), len(prompt))
			if logger.IsTrace() {
				for i, v := range instructions {
					logger.Debugf("instructions[%v]: %s\n", i, v)
				}
			}

			// call next
			if err := next.Serve(req, resp); err != nil {
				return err
			}
			return nil
		})
	}
}
