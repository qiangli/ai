package swarm

import (
	"os"
	"testing"

	// "github.com/google/uuid"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/llm/adapter"
	"github.com/qiangli/ai/swarm/util/calllog"
	"github.com/qiangli/ai/swarm/util/conf"
	hist "github.com/qiangli/ai/swarm/util/history"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

func defaultVars(cfg *api.App) (*api.Vars, error) {

	var wsbase = cfg.Base + "/workdir"
	if err := os.MkdirAll(wsbase, 0755); err != nil {
		return nil, err
	}

	var user = &api.User{
		Email: cfg.UserID,
	}
	var adapters = adapter.GetAdapters()

	var secrets = conf.LocalSecrets
	lfs, err := vfs.NewLocalFS([]string{wsbase})
	if err != nil {
		return nil, err
	}
	los, err := vos.NewLocalSystem(lfs)
	if err != nil {
		return nil, err
	}

	// dc, err := conf.Load(cfg.Base)
	// if err != nil {
	// 	return nil, err
	// }
	roots := &api.Roots{
		Workspace: &api.Root{
			Path: wsbase,
		},
	}
	dc := &api.DHNTConfig{
		Roots: roots,
	}

	assets, err := conf.Assets(dc)
	if err != nil {
		return nil, err
	}
	// blobs, err := conf.NewBlobs(dc, "")
	// if err != nil {
	// 	return nil, err
	// }

	tools, err := NewToolSystem(cfg.Base)
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

	var vars = &api.Vars{
		// ID:      uuid.NewString(),
		Global:  api.NewEnvironment(),
		Base:    cfg.Base,
		Roots:   roots,
		User:    user,
		Secrets: secrets,
		Assets:  assets,
		// Blobs:     blobs,
		Workspace: lfs,
		OS:        los,
		//
		Tools:    tools,
		Adapters: adapters,
		History:  mem,
		Log:      callogs,
	}
	return vars, nil
}

func TestTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	// home, _ := os.UserHomeDir()
	base := t.TempDir()
	cfg := &api.App{
		Base: base,
	}
	vars, err := defaultVars(cfg)
	// sw, err := New(vars)
	if err != nil {
		t.FailNow()
	}

	agent := &api.Agent{}
	agent.Runner = NewAgentToolRunner(vars, agent)

	tpl := atm.NewTemplate(vars, agent)

	var tools []*api.ToolFunc
	tools = append(tools, &api.ToolFunc{
		Type: api.ToolTypeAgent,
		Name: "ask_me",
		Kit:  "agent",
		// Agent: "ask",
	})
	// vars.Global.Set("__parent_agent", &api.Agent{
	// 	Name: "test",
	// 	Model: &api.Model{
	// 		Provider: "openai",
	// 		BaseUrl:  "https://api.openai.com/v1/",
	// 		ApiKey:   "openai",
	// 		Model:    "gpt-5-nano",
	// 	},
	// 	Tools: tools,
	// })

	text := `this is from ai: {{ai "@ask_me" "--log-level=verbose"  "tell me a joke"}}`
	data := map[string]any{}
	content, err := atm.CheckApplyTemplate(tpl, text, data)
	t.Logf("content: %v\n", content)
	if err != nil {
		t.Fatalf("%v", err)
	}
}
