package agent

import (
	_ "embed"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/api"
)

func RunAgent(cfg *api.AppConfig) error {
	log.Debugf("Agent: %s %s %v\n", cfg.Agent, cfg.Command, cfg.Args)

	in, err := GetUserInput(cfg)
	if err != nil {
		return err
	}

	if in.IsEmpty() {
		return internal.NewUserInputError("no query provided")
	}

	in.Agent = cfg.Agent
	in.Command = cfg.Command
	return RunSwarm(cfg, in)
}

func RunSwarm(cfg *api.AppConfig, input *api.UserInput) error {
	name := input.Agent
	command := input.Command
	log.Debugf("Running agent %q %s with swarm\n", name, command)

	vars, err := swarm.InitVars(cfg)
	if err != nil {
		return err
	}

	vars.History = cfg.History
	initLen := len(cfg.History)

	// TODO: this is for custom agent instruction defined in yaml
	vars.UserInput = input

	showInput(cfg, input)

	req := &api.Request{
		Agent:    name,
		Command:  command,
		RawInput: input,
	}
	resp := &api.Response{}

	sw := swarm.New(vars)

	if len(vars.History) > 0 {
		log.Infof("\033[33m⣿\033[0m recalling %v messages in memory less than %v minutes old\n", len(vars.History), cfg.MaxSpan)
	}

	if err := sw.Run(req, resp); err != nil {
		return err
	}

	log.Debugf("Agent %+v\n", resp.Agent)
	for _, m := range resp.Messages {
		log.Debugf("Message %+v\n", m)
	}

	var display = name
	if resp.Agent != nil {
		display = resp.Agent.Display
	}

	results := resp.Messages
	for _, v := range results {
		out := &api.Output{
			Display:     display,
			ContentType: v.ContentType,
			Content:     v.Content,
		}

		processOutput(cfg, out)

		cfg.Stdout = cfg.Stdout + v.Content
	}

	if len(vars.History) > initLen {
		if err := cfg.StoreHistory(vars.History[initLen:]); err != nil {
			log.Debugf("error saving history: %v", err)
		}
	}

	log.Debugf("Agent task completed: %s %v\n", cfg.Command, cfg.Args)

	return nil
}

func showInput(cfg *api.AppConfig, input *api.UserInput) {
	PrintInput(cfg, input)
}

func processOutput(cfg *api.AppConfig, message *api.Output) {
	switch message.ContentType {
	case api.ContentTypeText, "":
		processTextContent(cfg, message)
	case api.ContentTypeB64JSON:
		processImageContent(cfg, message)
	default:
		log.Debugf("Unsupported content type: %s\n", message.ContentType)
	}
}
