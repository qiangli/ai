package swarm

import (
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
	return sw, nil
}

func TestTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
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

	agent := &api.Agent{}

	tpl := sw.InitTemplate(agent)

	var tools []*api.ToolFunc
	tools = append(tools, &api.ToolFunc{
		Type:  api.ToolTypeAgent,
		Name:  "ask-me",
		Kit:   "agent",
		Agent: "ask",
	})
	sw.Vars.Global.Set("__parent_agent", &api.Agent{
		Name: "test",
		Model: &api.Model{
			Provider: "openai",
			BaseUrl:  "https://api.openai.com/v1/",
			ApiKey:   "openai",
			Model:    "gpt-5-nano",
		},
		Tools: tools,
	})

	text := `this is from ai: {{ai "@ask-me" "--log-level=verbose"  "tell me a joke"}}`
	data := map[string]any{}
	content, err := applyTemplate(tpl, text, data)
	t.Logf("content: %v\n", content)
	if err != nil {
		t.Fatalf("%v", err)
	}
}
