package atm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/atm/resource"
)

func (r *SystemKit) Cd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return "", err
	}
	return "", vars.RTE.OS.Chdir(dir)
}

func (r *SystemKit) Pwd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.RTE.OS.Getwd()
}

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

// func (r *SystemKit) Apply(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
// 	v, ok := args["script"]
// 	if !ok {
// 		return "", fmt.Errorf("missing script file")
// 	}
// 	s := api.ToString(v)

// 	if v, err := LoadScript(vars.RTE.Workspace, s); err != nil {
// 		return "", err
// 	} else {
// 		s = string(v)
// 	}

// 	var data = make(map[string]any)
// 	maps.Copy(data, vars.Global.GetAllEnvs())
// 	if vars.RootAgent.Environment != nil {
// 		maps.Copy(data, vars.RootAgent.Environment.GetAllEnvs())
// 	}
// 	maps.Copy(data, args)

// 	result, err := CheckApplyTemplate(vars.RootAgent.Template, s, data)
// 	if err != nil {
// 		return "", err
// 	}
// 	return api.ToString(result), nil
// }

func (r *SystemKit) Parse(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	result, err := conf.Parse(args["command"])
	if err != nil {
		return "", err
	}
	return api.ToString(result), nil
}

func (r *SystemKit) Format(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	format, _ := api.GetStrProp("format", args)
	if format == "" {
		format = "markdown"
	}

	var tpl = resource.FormatFile(format)

	return CheckApplyTemplate(vars.RootAgent.Template, tpl, args)
}
