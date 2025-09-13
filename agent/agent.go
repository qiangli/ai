package agent

import (
	_ "embed"
	"os"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
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

	//
	// if v, err := swarm.NewAgentCreator(cfg); err != nil {
	// 	return err
	// } else {
	// 	cfg.AgentCreator = v
	// }
	// cfg.AgentHandler = swarm.AgentHandler
	// cfg.ToolCaller = swarm.NewToolCaller(cfg)

	//
	if cfg.Env == nil {
		cfg.Env = make(map[string]string)
	}
	// app.Env["openai"] = os.Getenv("OPENAI_API_KEY")
	// app.Env["gemini"] = os.Getenv("GEMINI_API_KEY")
	// app.Env["anthropic"] = os.Getenv("ANTHROPIC_API_KEY")
	cfg.Env["OPENAI_API_KEY"] = os.Getenv("OPENAI_API_KEY")
	cfg.Env["GEMINI_API_KEY"] = os.Getenv("GEMINI_API_KEY")
	cfg.Env["ANTHROPIC_API_KEY"] = os.Getenv("ANTHROPIC_API_KEY")

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

	// TODO output as funtion return value
	cfg.Stdout = ""

	for _, v := range resp.Messages {
		out := &api.Output{
			Display:     display,
			ContentType: v.ContentType,
			Content:     v.Content,
		}

		processOutput(cfg, out)

		cfg.Stdout = cfg.Stdout + v.Content
	}

	if len(vars.History) > initLen {
		log.Debugf("Saving conversation\n")
		if err := mem.Save(vars.History[initLen:]); err != nil {
			log.Errorf("error saving conversation history: %v", err)
		}
	}

	log.Debugf("Agent task completed: %s %v\n", cfg.Command, cfg.Args)
	return nil
}

func showInput(cfg *api.AppConfig, input *api.UserInput) {
	if log.IsTrace() {
		log.Debugf("input: %+v\n", input)
	}

	PrintInput(cfg, input)
}

func processOutput(cfg *api.AppConfig, message *api.Output) {
	if log.IsTrace() {
		log.Debugf("output: %+v\n", message)
	}

	switch message.ContentType {
	case api.ContentTypeText, "":
		processTextContent(cfg, message)
	case api.ContentTypeB64JSON:
		processImageContent(cfg, message)
	default:
		log.Debugf("Unsupported content type: %s\n", message.ContentType)
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
