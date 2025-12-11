package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/db"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/util/conf"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

var essentialEnv = []string{"PATH", "PWD", "HOME", "USER", "SHELL"}

func loadUser(base string) (*api.User, error) {
	p := filepath.Join(base, "user.json")
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

func storeUser(base string, user *api.User) error {
	p := filepath.Join(base, "user.json")
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

func RunSwarm(cfg *api.App, user *api.User, argv []string) error {
	ctx := context.Background()

	// level := api.ToLogLevel(cfg.Arguments["log_level"])
	// log.GetLogger(ctx).SetLogLevel(level)
	// log.GetLogger(ctx).Debugf("Config: %+v\n", cfg)
	// logger := log.GetLogger(ctx)

	swarm.ClearAllEnv(essentialEnv)

	// var msg = cfg.Message

	mem, err := db.OpenMemoryStore(cfg.Base)
	if err != nil {
		return err
	}
	defer mem.Close()

	var ws = cfg.Workspace

	// // agent
	// if cfg.Kit == "agent" {
	// 	if cfg.Name != "" {
	// 		user.SetAgent(cfg.Name)
	// 		storeUser(cfg, user)
	// 	} else {
	// 		cfg.Name = user.Agent()
	// 	}
	// }

	var adapters = adapter.GetAdapters()

	var secrets = conf.LocalSecrets

	tmpdir := os.TempDir()
	project, _ := os.Getwd()
	dirs := []string{ws, project, tmpdir}
	roots := api.Roots{
		{Name: "Workspace", Path: ws},
		{Name: "Project Base", Path: project},
		{Name: "Temp Folder", Path: tmpdir},
	}
	allowedDirs, err := ResolvePaths(dirs)
	if err != nil {
		return err
	}
	lfs, _ := vfs.NewLocalFS(allowedDirs)
	los, _ := vos.NewLocalSystem(lfs)

	assets, err := conf.Assets(cfg.Base)
	if err != nil {
		return err
	}
	blobs, err := conf.NewBlobs(cfg.Base, "")
	if err != nil {
		return err
	}

	var rte = &api.ActionRTEnv{
		Base:      cfg.Base,
		Roots:     roots,
		User:      user,
		Secrets:   secrets,
		Workspace: lfs,
		OS:        los,
	}

	var tools = swarm.NewToolSystem(rte)

	sw := &swarm.Swarm{
		ID:       uuid.NewString(),
		User:     user,
		Secrets:  secrets,
		Assets:   assets,
		Tools:    tools,
		Adapters: adapters,
		Blobs:    blobs,
		//
		OS:        los,
		Workspace: lfs,
		History:   mem,
	}

	sw.Init(rte)

	sw.Vars.Global.Set("workspace", cfg.Workspace)
	// sw.Vars.Global.Set("query", msg)

	//
	// if cfg.HasInput() {
	// 	showInput(ctx, cfg)
	// }

	var out *api.Output
	if v, err := sw.Execv(ctx, argv); err != nil {
		// return err
		out = &api.Output{
			Content: fmt.Sprintf("❌ %+v", err),
		}
	} else {
		out = &api.Output{
			// Display:     v.Agent,
			ContentType: v.MimeType,
			Content:     v.Value,
		}
	}

	//
	processOutput(ctx, "markdown", out)

	// logger.Debugf("Agent task completed\n")

	return nil
}

func showInput(ctx context.Context, cfg *api.AppConfig) {
	if log.GetLogger(ctx).IsTrace() {
		log.GetLogger(ctx).Debugf("input: %+v\n", cfg.Message)
	}

	PrintInput(ctx, cfg)
}

func processOutput(ctx context.Context, format string, message *api.Output) {
	if log.GetLogger(ctx).IsTrace() {
		log.GetLogger(ctx).Debugf("output: %+v\n", message)
	}

	switch message.ContentType {
	case api.ContentTypeImageB64:
		var imageFile = filepath.Join(os.TempDir(), "image.png")
		processImageContent(ctx, imageFile, message)
	default:
		processTextContent(ctx, format, message)
	}
}
