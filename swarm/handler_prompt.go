package swarm

import (
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// System role prompt
func InstructionMiddleware(sw *Swarm) api.Middleware {
	resolve := func(parent *api.Agent, req *api.Request, s string) (string, error) {
		at, found := parseAgentCommand(s)
		if !found {
			return s, nil
		}
		out, err := sw.callAgent(parent, req, at.Name, at.Message)
		if err != nil {
			return "", err
		}

		return out, nil
	}
	return func(agent *api.Agent, next Handler) Handler {
		return HandlerFunc(func(req *api.Request, resp *api.Response) error {
			logger := log.GetLogger(req.Context())
			logger.Debugf("🔗 (instruction): %s\n", agent.Name)

			env := sw.globalEnv()

			var instructions []string

			add := func(in string) error {
				content, err := applyGlobal(agent.Template, "", in, env)
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
			addAll = func(a *api.Agent) error {
				for _, v := range a.Embed {
					return addAll(v)
				}
				in := a.Instruction()
				if in != "" {
					return add(in)
				}
				return nil
			}

			if err := addAll(agent); err != nil {
				return err
			}
			instruction := strings.Join(instructions, "\n")
			req.SetInstruction(instruction)

			logger.Debugf("instructions (%v): %s (%v)\n", len(instructions), abbreviate(instruction, 64), len(instruction))
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
