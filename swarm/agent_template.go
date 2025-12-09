package swarm

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/shell/tool/sh"
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

	//
	fm["fence"] = func() string {
		return "```"
	}

	// custom
	// ai
	fm["ai"] = func(args ...string) string {
		ctx := context.Background()

		at, err := conf.ParseActionArgs(args)
		if err != nil {
			return err.Error()
		}
		id := at.Kitname().ID()

		data, err := agent.Runner.Run(ctx, id, at)

		if err != nil {
			return err.Error()
		}
		result := api.ToResult(data)
		if result == nil {
			return ""
		}
		return result.Value
	}

	// core utils
	var core = []string{
		"base64",
		"basename",
		"cat",
		// "chmod",
		// "cp",
		"date",
		"dirname",
		"find",
		"gzip",
		"head",
		"ls",
		// "mkdir",
		// "mktemp",
		// "mv",
		// "rm",
		"shasum",
		"tac",
		"tail",
		"tar",
		"touch",
		"wget",
		"xargs",
	}
	for _, cmd := range core {
		fm[cmd] = func(args ...string) string {
			return RunCoreUtil(sw, cmd, args)
		}
	}

	return template.New("swarm").Funcs(fm)
}

func RunCoreUtil(sw *Swarm, cmd string, a []string) string {
	var args []string
	args = append(args, cmd)
	args = append(args, a...)

	var b bytes.Buffer
	ioe := &sh.IOE{Stdin: strings.NewReader(""), Stdout: &b, Stderr: &b}
	vs := sh.NewVirtualSystem(sw.OS, sw.Workspace, ioe)
	done, err := sh.RunCoreUtils(context.Background(), vs, args)
	if err != nil {
		return err.Error()
	}
	if !done {
		return "invalid/unsupported command: " + cmd
	}
	return b.String()
}
