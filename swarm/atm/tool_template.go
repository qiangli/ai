package atm

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/atm"
	"github.com/qiangli/ai/swarm/atm/conf"
	// "github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/shell/tool/sh"
)

// https://pkg.go.dev/text/template
// https://masterminds.github.io/sprig/
// https://github.com/golang/go/issues/18221
func NewFuncMap(vars *api.Vars) template.FuncMap {
	var fm = sprig.FuncMap()

	// overridge sprig
	fm["user"] = func() *api.User {
		return vars.User
		// return sw.User
	}
	// OS
	getenv := func(keys ...string) string {
		var envs []string
		if len(keys) == 0 {
			for k, v := range vars.Global.GetAllEnvs() {
				envs = append(envs, fmt.Sprintf("%s=%v", k, v))
			}
			sort.Strings(envs)
			return strings.Join(envs, "\n")
		}
		for _, k := range keys {
			v, ok := vars.Global.Get(k)
			if !ok {
				envs = append(envs, "")
			}
			envs = append(envs, api.ToString(v))
		}
		return fmt.Sprintf("%s", strings.Join(envs, "\n"))
	}
	fm["env"] = getenv
	fm["printenv"] = getenv
	setenv := func(key string, val any) string {
		if key == "" {
			return ""
		}
		vars.Global.Set(key, val)
		return ""
	}
	fm["setenv"] = setenv
	fm["count"] = count
	//
	fm["expandenv"] = func(s string) string {
		// bash name is leaked with os.Expand but ok.
		// bash is replaced with own that supports executing agent/tool
		// return os.Expand(s, getenv)
		return fmt.Sprintf("not supported. use golang template: {{.%s}}", s)
	}
	// Network:
	fm["getHostByName"] = func() string {
		return "localhost"
	}

	//
	fm["fence"] = func() string {
		return "```"
	}

	fm["encodeMD"] = encodeMD
	fm["decodeMD"] = decodeMD

	// custom

	// core utils
	// var core = []string{
	// 	"base64",
	// 	"basename",
	// 	"cat",
	// 	// "chmod",
	// 	// "cp",
	// 	"date",
	// 	"dirname",
	// 	"find",
	// 	// "gzip",
	// 	"head",
	// 	"ls",
	// 	// "mkdir",
	// 	// "mktemp",
	// 	// "mv",
	// 	// "rm",
	// 	"shasum",
	// 	"sleep",
	// 	// "tac",
	// 	"tail",
	// 	// "tar",
	// 	"time",
	// 	// "touch",
	// 	"wget",
	// 	"xargs",
	// }
	core := sh.CoreUtilsCommands

	for _, cmd := range core {
		fm[cmd] = func(args ...string) string {
			return RunCoreUtil(vars, cmd, args)
		}
	}
	return fm
}

func NewTemplate(vars *api.Vars, agent *api.Agent) *template.Template {
	fm := NewFuncMap(vars)

	// ai
	fm["ai"] = func(args ...string) string {
		if agent == nil {
			return "template: missing agent"
		}

		ctx := context.Background()
		at, err := conf.ParseActionArgs(args)
		if err != nil {
			return err.Error()
		}

		in := BuildEffectiveArgs(vars, agent, at)

		// result, err := sw.execm(ctx, agent, in)
		result, err := api.Exec(ctx, agent.Runner, in)

		if err != nil {
			return err.Error()
		}
		if result == nil {
			return ""
		}
		return result.Value
	}

	fm["asset"] = func(args ...string) string {
		if len(args) == 0 {
			return "pathname required. asset <pathname>..."
		}
		if agent == nil || agent.Config == nil {
			return "template asset: missing agent/config"
		}

		content, err := LoadAsset(agent.Config.Store, agent.Config.BaseDir, args...)
		if err != nil {
			return err.Error()
		}
		return encodeMD(content)
	}

	return template.New("swarm-agent").Funcs(fm)
}

func NewToolTemplate(vars *api.Vars, runner api.ActionRunner, tf *api.ToolFunc) *template.Template {
	fm := NewFuncMap(vars)

	// ai
	fm["ai"] = func(args ...string) string {
		if tf == nil {
			return "template: missing tool"
		}
		//
		ctx := context.Background()

		// log.GetLogger(ctx).Debugf("template tool: %s:%s args: %+v\n", tf.Kit, tf.Name, args)

		at, err := conf.ParseActionArgs(args)
		if err != nil {
			return err.Error()
		}

		in := BuildEffectiveParamArgs(vars, tf.Parameters, tf.Arguments, at)

		//
		result, err := api.Exec(ctx, runner, in)

		if err != nil {
			return err.Error()
		}
		if result == nil {
			return ""
		}
		return result.Value
	}

	fm["asset"] = func(args ...string) string {
		if len(args) == 0 {
			return "pathname required. asset <pathname>..."
		}
		if tf == nil || tf.Config == nil {
			return "template asset: missing tool/config"
		}
		content, err := LoadAsset(tf.Config.Store, tf.Config.BaseDir, args...)
		if err != nil {
			return err.Error()
		}
		return encodeMD(content)
	}

	return template.New("swarm-tool").Funcs(fm)
}

func RunCoreUtil(vars *api.Vars, cmd string, a []string) string {
	var args []string
	args = append(args, cmd)
	args = append(args, a...)

	var b bytes.Buffer
	ioe := &sh.IOE{Stdin: strings.NewReader(""), Stdout: &b, Stderr: &b}

	vs := sh.NewVirtualSystem(vars.OS, vars.Workspace, ioe)
	done, err := sh.RunCoreUtils(context.Background(), vs, args)
	if err != nil {
		return err.Error()
	}
	if !done {
		return "invalid/unsupported command: " + cmd
	}
	return b.String()
}

func LoadAsset(store api.AssetStore, base string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("Missing filename")
	}
	if as, ok := store.(api.AssetFS); ok {
		var content string
		for _, name := range args {
			v, err := as.ReadFile(path.Join(base, name))
			if err != nil {
				return "", err
			}
			content += string(v)
		}
		return content, nil
	}
	if ws, ok := store.(api.Workspace); ok {
		var content string
		for _, name := range args {
			v, err := ws.ReadFile(path.Join(base, name), nil)
			if err != nil {
				return "", err
			}
			content += string(v)
		}
		return content, nil
	}

	return "", fmt.Errorf("Asset not supported. base: %s. files: %v", base, args)
}

func splitLines(text string) []string {
	return strings.Split(text, "\n")
}

func count(obj any) int {
	switch v := obj.(type) {
	case []byte: // uint8
		return len(v)
	case string:
		return len(splitLines(v))
	case []int, []int8, []int16, []int32, []int64,
		[]uint, []uint16, []uint32, []uint64,
		[]string, []float64, []float32, []struct{}:
		return reflect.ValueOf(obj).Len()
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64,
		complex64, complex128:
		// Return 1 for single value
		return 1
	case struct{}:
		return reflect.TypeOf(obj).NumField()
	default:
		return 0
	}
}

// https://github.com/golang/go/issues/18221
// UNICODE U+00060
// HEX CODE &#x60;
// HTML CODE &#96;
// HTML ENTITY &grave;
func encodeMD(s string) string {
	return strings.ReplaceAll(s, "`", "&grave;")
}
func decodeMD(s string) string {
	return strings.ReplaceAll(s, "&grave;", "`")
}
