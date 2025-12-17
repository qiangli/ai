package atm

import (
	"context"
	"fmt"
	"maps"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"
)

// no-op tool that does nothing
func (r *SystemKit) Pass(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return "", nil
}

// func (r *SystemKit) Cd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	dir, err := api.GetStrProp("dir", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return "", vars.RTE.OS.Chdir(dir)
// }

// func (r *SystemKit) Pwd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	return vars.RTE.OS.Getwd()
// }

func (r *SystemKit) Exec(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	cmd, err := api.GetStrProp("command", args)
	if err != nil {
		return "", err
	}
	if len(cmd) == 0 {
		return "", fmt.Errorf("command is empty")
	}
	// command := argv[0]
	// rest := argv[1:]
	vs := vars.RTE.OS
	result, err := ExecCommand(ctx, vs, vars, cmd, nil)

	if err != nil {
		return "", err
	}
	return api.ToString(result), nil
}

func (r *SystemKit) Bash(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	// shell handles command/script if empty
	result, err := vars.RootAgent.Shell.Run(ctx, "", args)
	if err != nil {
		return "", err
	}
	return api.ToString(result), nil
}

func (r *SystemKit) Apply(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	tpl, err := api.GetStrProp("template", args)
	if err != nil {
		return "", err
	}
	if v, err := LoadURIContent(vars.RTE.Workspace, tpl); err != nil {
		return "", err
	} else {
		tpl = string(v)
	}

	var data = make(map[string]any)
	maps.Copy(data, vars.Global.GetAllEnvs())
	maps.Copy(data, args)

	return CheckApplyTemplate(vars.RootAgent.Template, tpl, data)
}

func (r *SystemKit) Parse(ctx context.Context, vars *api.Vars, name string, args map[string]any) (api.ArgMap, error) {
	result, err := conf.Parse(args["command"])
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *SystemKit) Format(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	var tpl string
	tpl, _ = api.GetStrProp("template", args)
	if tpl != "" {
		if v, err := LoadURIContent(vars.RTE.Workspace, tpl); err != nil {
			return "", err
		} else {
			tpl = string(v)
		}
	}
	if tpl == "" {
		format, _ := api.GetStrProp("format", args)
		if format == "" {
			format = "markdown"
		}
		tpl = resource.FormatFile(format)
	}

	var data = make(map[string]any)
	maps.Copy(data, vars.Global.GetAllEnvs())
	maps.Copy(data, args)

	return CheckApplyTemplate(vars.RootAgent.Template, tpl, data)
}

func (r *SystemKit) GetToolConfig(ctx context.Context, vars *api.Vars, tf string, args map[string]any) (string, error) {
	tid, err := api.GetStrProp("tool", args)
	if err != nil {
		return "", err
	}

	kit, name := api.Kitname(tid).Decode()

	tc, err := vars.RTE.Assets.FindToolkit(vars.RTE.User.Email, kit)
	if err != nil {
		return "", err
	}

	if tc != nil {
		for _, v := range tc.Tools {
			if v.Name == name {
				// params, err := json.Marshal(v.Parameters)
				// if err != nil {
				// 	return "", err
				// }
				// TODO params may need better handling
				// log.GetLogger(ctx).Debugf("Tool info: %s %+v\n", tid, string(params))
				// return fmt.Sprintf(tpl, kit, v.Name, v.Description, string(params)), nil
				args["config"] = tc
				return string(tc.RawContent), nil
			}
		}
	}
	return "", fmt.Errorf("unknown tool: %s", tid)
}
