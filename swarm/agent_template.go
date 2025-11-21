package swarm

import (
	"context"
	"fmt"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
)

// https://pkg.go.dev/text/template
// https://masterminds.github.io/sprig/
func NewTemplate(sw *Swarm, agent *api.Agent) *template.Template {
	var fm = sprig.FuncMap()

	// overridge sprig
	fm["user"] = func() *api.User {
		return sw.User
	}
	// OS
	getenv := func(key string) string {
		v, ok := sw.Vars.Global.Get(key)
		if !ok {
			return ""
		}
		if s, ok := v.(string); ok {
			return s
		}
		return fmt.Sprintf("%v", v)
	}
	fm["env"] = getenv
	fm["expandenv"] = func(s string) string {
		// bash name is leaked with os.Expand but ok.
		// bash is replaced with own that supports executing agent/tool
		return os.Expand(s, getenv)
	}
	// Network:
	fm["getHostByName"] = func() string {
		return "localhost"
	}

	// ai
	fm["ai"] = func(args ...string) string {
		ctx := context.Background()

		at, err := conf.ParseActionArgs(args)
		if err != nil {
			return err.Error()
		}
		id := api.KitName(at.Kit + ":" + at.Name).ID()

		data, err := agent.Runner.Run(ctx, id, at.Arguments)

		if err != nil {
			return err.Error()
		}
		result := api.ToResult(data)
		if result == nil {
			return ""
		}
		return result.Value
	}

	return template.New("swarm").Funcs(fm)
}
