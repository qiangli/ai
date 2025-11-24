package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
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

func loadUser(cfg *api.AppConfig) (*api.User, error) {
	p := filepath.Join(cfg.Base, "user.json")
	file, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var user api.User
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func storeUser(cfg *api.AppConfig, user *api.User) error {
	p := filepath.Join(cfg.Base, "user.json")
	file, err := os.Create(p)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	err = encoder.Encode(user)
	if err != nil {
		return err
	}

	return nil
}

func RunSwarm(ctx context.Context, cfg *api.AppConfig, input *api.UserInput) error {
	logger := log.GetLogger(ctx)
	swarm.ClearAllEnv(essentialEnv)

	vars, err := InitVars(cfg)
	if err != nil {
		return err
	}

	mem, err := db.OpenMemoryStore(cfg)
	if err != nil {
		return err
	}
	defer mem.Close()

	showInput(ctx, cfg, input)

	var root = cfg.Workspace

	//
	var user *api.User
	who, _ := util.WhoAmI()
	if v, err := loadUser(cfg); err != nil {
		user = &api.User{
			Display:  who,
			Settings: make(map[string]any),
		}
	} else {
		user = v
		user.Display = who
	}
	if cfg.Name != "" {
		user.SetAgent(cfg.Name)
		storeUser(cfg, user)
	} else {
		cfg.Name = user.Agent()
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
		History:   mem,
	}

	sw.Init()

	var args = make(map[string]any)
	maps.Copy(args, cfg.ToMap())
	maps.Copy(args, input.Arguments)

	// initial query is required.
	if args["query"] == "" {
		return fmt.Errorf("%s: query missing", cfg.Name)
	}

	var out *api.Output
	if v, err := sw.Execm(ctx, cfg.Name, args); err != nil {
		return err
	} else {
		out = &api.Output{
			Display:     cfg.Name,
			ContentType: v.MimeType,
			Content:     v.Value,
		}
	}

	//
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
	case api.ContentTypeImageB64:
		var imageFile = filepath.Join(os.TempDir(), "image.png")
		processImageContent(ctx, imageFile, message)
	default:
		processTextContent(ctx, cfg, message)
	}
}

func InitVars(app *api.AppConfig) (*api.Vars, error) {
	var vars = api.NewVars()

	// global envs
	vars.Global.Set("workspace", app.Workspace)

	return vars, nil
}
