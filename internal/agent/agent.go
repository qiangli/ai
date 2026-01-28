package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/google/uuid"

	"github.com/qiangli/ai/internal/util"
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

func Run(argv []string) error {
	var app = &api.App{}
	var base string
	for i, v := range argv {
		if slices.Contains([]string{"--base", "-base"}, v) {
			if len(argv) > i+1 {
				base = argv[i+1]
				break
			}
		}
	}
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		base = filepath.Join(home, ".ai")
	}
	app.Base = base

	//
	var user *api.User
	who, _ := util.WhoAmI()
	app.UserID = who
	if v, err := loadUser(app.Base); err != nil {
		user = &api.User{
			Display:  who,
			Settings: make(map[string]any),
		}
	} else {
		user = v
		user.Display = who
	}

	if err := RunSwarm(app, user, argv); err != nil {
		return err
	}
	return nil
}

func RunSwarm(cfg *api.App, user *api.User, args []string) error {
	ctx := context.Background()

	// init
	sw, err := initSwarm(ctx, cfg, user)
	if err != nil {
		return err
	}

	// ***
	// parse input
	// initial pass
	argm, err := sw.Parse(ctx, args)
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
	// mirror console level to tee (file) outputs so file contains same verbosity
	logger.SetTeeLogLevel(level)
	logger.Debugf("Config: %+v\n", cfg)

	// ***
	// perform action
	var out = api.Output{
		Display: "",
	}

	id := argm.Kitname().ID()
	if id == "" {
		// default @root/root
		argm["kit"] = "agent"
		argm["pack"] = "root"
		argm["name"] = "root"
	}
	if v, err := sw.Exec(ctx, argm); err != nil {
		// return err
		out.Content = fmt.Sprintf("❌ %+v", err)
	} else {
		out.ContentType = v.MimeType
		out.Content = v.Value
	}

	// console outpu
	format := argm.GetString("format")
	if format == "" {
		format = "markdown"
	}
	processOutput(ctx, format, &out)

	/* close tee file if opened */
	if logger != nil {
		if err := logger.CloseTee(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close tee file: %v\n", err)
		}
	}

	return nil
}

func initSwarm(ctx context.Context, cfg *api.App, user *api.User) (*swarm.Swarm, error) {
	var sessionID = api.SessionID(uuid.NewString())

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

	//
	callDir := filepath.Join(roots.Workspace.Path, "var", "log", "toolcall")
	teeDir := filepath.Join(roots.Workspace.Path, "var", "log", "conversation")

	//

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
	callogs, err := calllog.NewFileCallLog(callDir, sessionID)
	if err != nil {
		return nil, err
	}

	tools, err := swarm.NewToolSystem(cfg.Base)
	if err != nil {
		return nil, err
	}

	var vars = &api.Vars{
		SessionID: sessionID,
		//
		Base:      cfg.Base,
		Workspace: lfs,
		User:      user,
		Secrets:   secrets,
		OS:        los,
		//
		Roots:  roots,
		Assets: assets,
		Blobs:  blobs,
		//
		Tools:    tools,
		Adapters: adapters,
		History:  mem,
		Log:      callogs,
	}

	sw, err := swarm.New(vars)
	if err != nil {
		return nil, err
	}

	// hook up tee logging for this run: write to <workspace>/var/log/conversation/<uuid>.log
	if err := os.MkdirAll(teeDir, 0o755); err == nil {
		teeFile := filepath.Join(teeDir, string(sessionID)+".log")
		logger := log.GetLogger(ctx)
		if err := logger.SetTeeFile(teeFile); err != nil {
			// best effort: if setting tee fails, log to stderr
			fmt.Fprintf(os.Stderr, "failed to set tee file %s: %v\n", teeFile, err)
		} else {
			// default tee log level mirrors console unless changed later
			logger.SetTeeLogLevel(api.Informative)
		}
	} else {
		fmt.Fprintf(os.Stderr, "failed to create tee dir %s: %v\n", teeDir, err)
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
