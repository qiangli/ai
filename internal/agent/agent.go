package agent

import (
	"context"
	_ "embed"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/db"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/util/conf"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

func RunAgent(ctx context.Context, app *api.AppConfig) error {
	in, err := GetUserInput(ctx, app)
	if err != nil {
		return err
	}

	return RunSwarm(ctx, app, in)
}

var essentialEnv = []string{"PATH", "PWD", "HOME", "USER", "SHELL"}

func RunSwarm(ctx context.Context, cfg *api.AppConfig, input *api.UserInput) error {
	logger := log.GetLogger(ctx)
	swarm.ClearAllEnv(essentialEnv)

	// name := cfg.Name
	// if name == "" {
	// 	name = "agent"
	// }

	// logger.Debugf("Running agent %q\n", name)

	vars, err := InitVars(cfg)
	if err != nil {
		return err
	}

	// mem := NewFileMemStore(cfg)
	mem, err := db.OpenMemoryStore(cfg)
	if err != nil {
		return err
	}
	defer mem.Close()

	showInput(ctx, cfg, input)

	req := api.NewRequest(ctx, cfg.Name, input)
	resp := &api.Response{}

	var root = cfg.Workspace

	who, _ := util.WhoAmI()
	var user = &api.User{
		Display: who,
	}
	var adapters = adapter.GetAdapters()

	var secrets = conf.LocalSecrets
	var lfs = vfs.NewLocalFS(root)
	var los = vos.NewLocalSystem(root)

	assets, err := conf.Assets(cfg)
	if err != nil {
		return err
	}
	blobs, err := conf.NewBlobs(cfg, "")
	if err != nil {
		return err
	}
	var tools = swarm.NewToolSystem(root, user, secrets, assets, lfs, los)

	sw := &swarm.Swarm{
		ID:        uuid.NewString(),
		Root:      root,
		Vars:      vars,
		User:      user,
		Secrets:   secrets,
		Assets:    assets,
		Tools:     tools,
		Adapters:  adapters,
		Blobs:     blobs,
		OS:        los,
		Workspace: lfs,
		//
		History: mem,
	}

	sw.Init()

	// TODO remove error return from Run?
	if err := sw.Run(req, resp); err != nil {
		// return err
		resp.Result = &api.Result{
			Value: err.Error(),
		}
	}

	logger.Debugf("Agent %+v\n", resp.Agent)
	if resp.Result != nil {
		logger.Debugf("Result content %s\n", resp.Result.Value)
	}
	for _, m := range resp.Messages {
		logger.Debugf("Message %+v\n", m)
	}

	var display = cfg.Name
	if resp.Agent != nil {
		display = resp.Agent.Display
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

	logger.Debugf("Agent task completed\n")
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
	// vars.New = app.New
	// vars.MaxTurns = app.MaxTurns
	// vars.MaxTime = app.MaxTime
	// vars.MaxHistory = app.MaxHistory
	// vars.Context = app.Context
	// vars.MaxSpan = app.MaxSpan
	// // vars.Message = app.Message
	// vars.Format = app.Format
	// vars.Models = app.Models

	// vars.Unsafe = false
	// vars.DryRun = app.DryRun
	// vars.DryRunContent = app.DryRunContent
	//
	// vars.Workspace = app.Workspace

	return vars, nil
}
