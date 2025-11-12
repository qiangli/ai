package swarm

import (
	"strings"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

// System role prompt
func InstructionMiddlewareFunc(sw *Swarm) func(*api.Agent) api.Middleware {
	return func(agent *api.Agent) api.Middleware {
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
		return func(next Handler) Handler {
			return HandlerFunc(func(req *api.Request, resp *api.Response) error {
				logger := log.GetLogger(req.Context())
				logger.Debugf("🔗 (instruction): %s\n", agent.Name)

				env := sw.globalEnv()

				var instructions []string

				add := func(in *api.Instruction, sender string) error {
					content, err := sw.applyGlobal(in.Type, in.Content, env)
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
					if a.Instruction != nil {
						return add(a.Instruction, a.Name)
					}
					return nil
				}

				if err := addAll(agent); err != nil {
					return err
				}

				req.Instruction = strings.Join(instructions, "\n")

				logger.Debugf("instructions (%v): %s (%v)\n", len(instructions), abbreviate(req.Instruction, 64), len(req.Instruction))
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
}
