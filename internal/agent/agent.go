package agent

import (
	"strings"

	"github.com/qiangli/ai/internal"
	// "github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
)

// type UserInput = api.Request
// type ChatMessage = api.Response

func agentList() map[string]string {
	return AgentDesc
}

func agentCommandList() map[string]string {
	return AgentCommands
}

func HandleCommand(cfg *internal.AppConfig) error {
	log.Debugf("HandleCommand: %s %v\n", cfg.Command, cfg.Args)

	cmd := cfg.Command

	if cmd != "" {
		// $ ai /command
		if strings.HasPrefix(cmd, "/") {
			name := strings.TrimSpace(cmd[1:])
			in, err := GetUserInput(cfg)
			if err != nil {
				return err
			}

			if name == "" && in.IsEmpty() {
				return internal.NewUserInputError("no command and message provided")
			}

			in.Agent = "script"
			in.Subcommand = name
			return handleAgent(cfg, in)
		}

		// $ ai @agent
		if strings.HasPrefix(cmd, "@") {
			name := strings.TrimSpace(cmd[1:])
			if name == "" {
				// auto select
				// $ ai @ message...
				return AgentHelp(cfg)
			}

			na := strings.SplitN(name, "/", 2)
			agent := na[0]
			// if !hasAgent(agent) {
			// 	return internal.NewUserInputError("no such agent: " + agent)
			// }
			in, err := GetUserInput(cfg)
			if err != nil {
				return err
			}
			if in.IsEmpty() {
				return internal.NewUserInputError("no message content")
			}

			in.Agent = agent
			if len(na) > 1 {
				in.Subcommand = na[1]
			}
			return handleAgent(cfg, in)
		}
	}

	// auto select the best agent to handle the user query if there is message content
	// $ ai message...
	return AgentHelp(cfg)
}
