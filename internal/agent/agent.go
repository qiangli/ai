package agent

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent/conf"
	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/vfs"
	"github.com/qiangli/ai/swarm/vos"
)

func RunAgent(ctx context.Context, app *api.AppConfig) error {
	log.GetLogger(ctx).Debugf("Agent: %s %v\n", app.Agent, app.Args)
	in, err := GetUserInput(ctx, app)
	if err != nil {
		return err
	}

	if in.IsEmpty() && app.Message == "" {
		return internal.NewUserInputError("no query provided")
	}

	// in.Agent = app.Agent

	return RunSwarm(ctx, app, in)
}

func RunSwarm(ctx context.Context, cfg *api.AppConfig, input *api.UserInput) error {
	name := cfg.Agent
	if name == "" {
		name = "agent"
	}

	log.GetLogger(ctx).Debugf("Running agent %q with swarm\n", name)

	vars, err := InitVars(cfg)
	if err != nil {
		return err
	}
	// History
	// preload - may not be used depending on context agent
	mem := NewFileMemStore(cfg)
	history, err := mem.Load(&api.MemOption{
		MaxHistory: cfg.MaxHistory,
		MaxSpan:    cfg.MaxSpan,
	})
	if err != nil {
		return err
	}
	// TODO depends on new/max-history,max-span/context flags
	// if len(vars.History) > 0 {
	// 	log.GetLogger(ctx).Infof("⣿ recalling %v messages in memory less than %v minutes old\n", len(vars.History), cfg.MaxSpan)
	// }

	initLen := len(history)
	vars.History = history

	showInput(ctx, cfg, input)

	req := &api.Request{
		Agent:    name,
		RawInput: input,
	}
	resp := &api.Response{}

	var user = &api.User{}
	var secrets = conf.LocalSecrets
	var adapters = adapter.GetAdapters()
	var lfs = vfs.NewLocalFS()
	var los = vos.NewLocalSystem()
	assets, err := conf.Assets(cfg)
	if err != nil {
		return err
	}
	blobs, err := conf.NewBlobs(cfg, "")
	if err != nil {
		return err
	}
	var tools = swarm.NewToolSystem(user, secrets, assets, lfs, los)

	sw := &swarm.Swarm{
		Vars:     vars,
		User:     user,
		Secrets:  secrets,
		Assets:   assets,
		Tools:    tools,
		Adapters: adapters,
		Blobs:    blobs,
		OS:       los,
		FS:       lfs,
		//
		History: mem,
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
	// cfg.Stdout = ""

	// for _, v := range resp.Messages {
	// 	out := &api.Output{
	// 		Display:     display,
	// 		ContentType: v.ContentType,
	// 		Content:     v.Content,
	// 	}

	// 	processOutput(ctx, cfg, out)
	// 	// cfg.Stdout = cfg.Stdout + v.Content
	// }

	if len(vars.History) > initLen {
		log.GetLogger(ctx).Debugf("Saving conversation\n")
		if err := mem.Save(vars.History[initLen:]); err != nil {
			log.GetLogger(ctx).Errorf("error saving conversation history: %v", err)
		}
	}

	var out *api.Output
	if resp.Result != nil {
		out = &api.Output{
			Display:     display,
			ContentType: resp.Result.MimeType,
			Content:     resp.Result.Value,
		}
	} else {
		if len(resp.Messages) > 0 {
			msg := resp.Messages[len(resp.Messages)-1]
			out = &api.Output{
				Display:     display,
				ContentType: msg.ContentType,
				Content:     msg.Content,
			}
		}
	}
	processOutput(ctx, cfg, out)

	log.GetLogger(ctx).Debugf("Agent task completed: %v\n", cfg.Args)
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
	// case api.ContentTypeText, "":
	// 	processTextContent(ctx, cfg, message)
	case api.ContentTypeImageB64:
		var imageFile = filepath.Join(os.TempDir(), "image.png")
		processImageContent(ctx, imageFile, message)
	default:
		processTextContent(ctx, cfg, message)
	}
}

func InitVars(app *api.AppConfig) (*api.Vars, error) {
	var vars = api.NewVars()

	// Setting configuration values from the app to vars
	vars.LogLevel = api.ToLogLevel(app.LogLevel)
	vars.ChatID = app.ChatID
	vars.New = app.New
	vars.MaxTurns = app.MaxTurns
	vars.MaxTime = app.MaxTime
	vars.MaxHistory = app.MaxHistory
	vars.Context = app.Context
	vars.MaxSpan = app.MaxSpan
	// vars.Message = app.Message
	vars.Format = app.Format
	vars.Models = app.Models
	vars.Unsafe = app.Unsafe
	vars.DryRun = app.DryRun
	vars.DryRunContent = app.DryRunContent
	//
	vars.Workspace = app.Workspace
	//
	vars.Workspace = app.Workspace

	return vars, nil
}
