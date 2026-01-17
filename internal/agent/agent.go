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

	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/util/calllog"
	"github.com/qiangli/ai/swarm/util/conf"
	hist "github.com/qiangli/ai/swarm/util/history"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

var essentialEnv = []string{"PATH", "PWD", "HOME", "USER", "SHELL", "GOPATH"}

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

// func storeUser(base string, user *api.User) error {
// 	p := filepath.Join(base, "user.json")
// 	file, err := os.Create(p)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	encoder := json.NewEncoder(file)
// 	err = encoder.Encode(user)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

func RunSwarm(cfg *api.App, user *api.User, argv []string) error {
	ctx := context.Background()

	// init
	sw, err := initSwarm(ctx, cfg, user)
	if err != nil {
		return err
	}

	// ***
	// parse input
	// initial pass
	argm, err := sw.Parse(ctx, argv)
	if err != nil {
		return err
	}

	// default from user preference. update only if not set
	for k, v := range user.Settings {
		if _, ok := argm[k]; !ok {
			argm[k] = v
		}
	}

	// show input
	level := api.ToLogLevel(argm["log_level"])
	logger := log.GetLogger(ctx)
	logger.SetLogLevel(level)
	logger.Debugf("Config: %+v\n", cfg)

	// ***
	// perform action
	var out = api.Output{
		Display: "",
	}

	if msg, ok := argm["message"]; ok {
		sw.Vars.Global.Set("message", msg)
	}

	id := argm.Kitname().ID()
	if id != "" {
		if v, err := sw.Exec(ctx, argm); err != nil {
			// return err
			out.Content = fmt.Sprintf("❌ %+v", err)
		} else {
			out.ContentType = v.MimeType
			out.Content = v.Value
		}
	}

	// console outpu
	format := argm.GetString("format")
	if format == "" {
		format = "markdown"
	}
	processOutput(ctx, format, &out)

	/*  */
	return nil
}

func initSwarm(ctx context.Context, cfg *api.App, user *api.User) (*swarm.Swarm, error) {
	swarm.ClearAllEnv(essentialEnv)

	var adapters = adapter.GetAdapters()
	var secrets = conf.LocalSecrets

	dc, err := conf.Load(cfg.Base)
	if err != nil {
		return nil, err
	}
	var roots = dc.Roots
	dirs, err := roots.AllowedDirs()
	if err != nil {
		return nil, err
	}
	if len(dirs) == 0 {
		return nil, fmt.Errorf("root directories not configed")
	}
	lfs, _ := vfs.NewLocalFS(dirs)
	los, _ := vos.NewLocalSystem(lfs)

	assets, err := conf.Assets(dc)
	if err != nil {
		return nil, err
	}
	blobs, err := conf.NewBlobs(dc, "")
	if err != nil {
		return nil, err
	}
	mem, err := hist.NewFileMemStore(roots.Workspace.Path)
	if err != nil {
		return nil, err
	}
	callogs, err := calllog.NewFileCallLog(roots.Workspace.Path)
	if err != nil {
		return nil, err
	}

	tools, err := swarm.NewToolSystem(cfg.Base)
	if err != nil {
		return nil, err
	}

	var rte = &api.ActionRTEnv{
		ID:        uuid.NewString(),
		Base:      cfg.Base,
		Roots:     roots,
		User:      user,
		Secrets:   secrets,
		Assets:    assets,
		Blobs:     blobs,
		Workspace: lfs,
		OS:        los,
		//
		Tools:    tools,
		Adapters: adapters,
		History:  mem,
		Log:      callogs,
	}

	sw := &swarm.Swarm{
		// ID:       uuid.NewString(),
		// User:     user,
		// Secrets:  secrets,
		// Assets:   assets,
		// Tools:    tools,
		// Adapters: adapters,
		// Blobs:    blobs,
		// //
		// OS:        los,
		// Workspace: lfs,
		// History:   mem,
		// Log:       callogs,
	}

	if err := sw.Init(rte); err != nil {
		return nil, err
	}

	return sw, nil
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
