package swarm

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/db"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/util/conf"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

func defaultSwarm(cfg *api.AppConfig) (*Swarm, error) {
	var vars = api.NewVars()

	var root = cfg.Workspace

	var user = &api.User{
		Email: cfg.User,
	}
	var adapters = adapter.GetAdapters()

	var secrets = conf.LocalSecrets
	var lfs = vfs.NewLocalFS(root)
	var los = vos.NewLocalSystem(root)

	assets, err := conf.Assets(cfg)
	if err != nil {
		return nil, err
	}
	blobs, err := conf.NewBlobs(cfg, "")
	if err != nil {
		return nil, err
	}
	var tools = NewToolSystem(root, user, secrets, assets, lfs, los)

	mem, err := db.OpenMemoryStore(cfg)
	if err != nil {
		return nil, err
	}
	sw := &Swarm{
		ChatID:    uuid.NewString(),
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
	return sw, nil
}

func TestTemplate(t *testing.T) {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".ai")
	cfg := &api.AppConfig{
		Base:      base,
		Workspace: "/tmp",
	}
	sw, err := defaultSwarm(cfg)
	if err != nil {
		t.FailNow()
	}

	// req := &api.Request{
	// 	Name: "ask",
	// 	RawInput: &api.UserInput{
	// 		Message: "",
	// 	},
	// }
	ctx := context.TODO()
	sw.initTemplate(ctx)
	// sw.createAgent(ctx, req)

	text := `this is from ai: {{ai "@ask" "--verbose" "what is the weather in dublin ca."}}`
	data := map[string]any{}
	content, err := sw.applyTemplate(text, data)
	if err != nil {
		t.Fatalf("%v", err)
	}
	t.Logf("%v", content)
}
