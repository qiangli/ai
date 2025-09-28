package agent

import (
	"context"
	_ "embed"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func RunAgent(ctx context.Context, cfg *api.AppConfig) error {
	// log.GetLogger(ctx).Debugf("Agent: %s %s %v\n", cfg.Agent, cfg.Command, cfg.Args)
	log.GetLogger(ctx).Debugf("Agent: %s %v\n", cfg.Agent, cfg.Args)

	in, err := GetUserInput(ctx, cfg)
	if err != nil {
		return err
	}

	if in.IsEmpty() && cfg.Message == "" {
		return internal.NewUserInputError("no query provided")
	}

	in.Agent = cfg.Agent
	// in.Command = cfg.Command

	return RunSwarm(ctx, cfg, in)
}

func RunSwarm(ctx context.Context, cfg *api.AppConfig, input *api.UserInput) error {
	name := input.Agent
	// command := input.Command
	log.GetLogger(ctx).Debugf("Running agent %q with swarm\n", name)

	//
	vars, err := InitVars(cfg)
	if err != nil {
		return err
	}

	// History
	mem := NewFileMemStore(cfg)
	history, err := mem.Load(&api.MemOption{
		MaxHistory: cfg.MaxHistory,
		MaxSpan:    cfg.MaxSpan,
	})
	if err != nil {
		return err
	}
	initLen := len(history)

	// TODO: this is for custom agent instruction defined in yaml
	vars.UserInput = input

	showInput(ctx, cfg, input)

	req := &api.Request{
		Agent: name,
		// Command:  command,
		RawInput: input,
	}
	resp := &api.Response{}

	sw := swarm.New(vars)

	if len(vars.History) > 0 {
		log.GetLogger(ctx).Infof("⣿ recalling %v messages in memory less than %v minutes old\n", len(vars.History), cfg.MaxSpan)
	}

	if err := sw.Run(req, resp); err != nil {
		return err
	}

	log.GetLogger(ctx).Debugf("Agent %+v\n", resp.Agent)
	for _, m := range resp.Messages {
		log.GetLogger(ctx).Debugf("Message %+v\n", m)
	}

	var display = name
	if resp.Agent != nil {
		display = resp.Agent.Display
	}

	// TODO output as funtion return value
	cfg.Stdout = ""

	for _, v := range resp.Messages {
		out := &api.Output{
			Display:     display,
			ContentType: v.ContentType,
			Content:     v.Content,
		}

		processOutput(ctx, cfg, out)

		cfg.Stdout = cfg.Stdout + v.Content
	}

	if len(vars.History) > initLen {
		log.GetLogger(ctx).Debugf("Saving conversation\n")
		if err := mem.Save(vars.History[initLen:]); err != nil {
			log.GetLogger(ctx).Errorf("error saving conversation history: %v", err)
		}
	}

	log.GetLogger(ctx).Debugf("Agent task completed: %v\n", cfg.Args)
	// 	log.GetLogger(ctx).Debugf("Agent task completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func showInput(ctx context.Context, cfg *api.AppConfig, input *api.UserInput) {
	if log.GetLogger(ctx).IsTrace() {
		log.GetLogger(ctx).Debugf("input: %+v\n", input)
	}

	PrintInput(ctx, cfg, input)
}

func processOutput(ctx context.Context, cfg *api.AppConfig, message *api.Output) {
	if log.GetLogger(ctx).IsTrace() {
		log.GetLogger(ctx).Debugf("output: %+v\n", message)
	}

	switch message.ContentType {
	case api.ContentTypeText, "":
		processTextContent(ctx, cfg, message)
	case api.ContentTypeB64JSON:
		processImageContent(ctx, cfg, message)
	default:
		log.GetLogger(ctx).Debugf("Unsupported content type: %s\n", message.ContentType)
	}
}

func InitVars(app *api.AppConfig) (*api.Vars, error) {
	var vars = api.NewVars()
	//
	vars.Config = app
	//
	vars.Workspace = app.Workspace
	// vars.Repo = app.Repo
	vars.Home = app.Home
	vars.Temp = app.Temp

	//
	sysInfo, err := util.CollectSystemInfo()
	if err != nil {
		return nil, err
	}

	vars.Arch = sysInfo.Arch
	vars.OS = sysInfo.OS
	vars.ShellInfo = sysInfo.ShellInfo
	vars.OSInfo = sysInfo.OSInfo
	vars.UserInfo = sysInfo.UserInfo

	return vars, nil
}
