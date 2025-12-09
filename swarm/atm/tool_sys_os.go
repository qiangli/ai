package atm

import (
	"context"
	// "fmt"
	// "net/url"

	// // "maps"
	// "strings"

	"github.com/qiangli/ai/swarm/api"
	// "github.com/qiangli/ai/swarm/atm/conf"
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
	command, err := api.GetStrProp("command", args)
	if err != nil {
		return "", err
	}
	// argv, err := api.GetArrayProp("arguments", args)
	// if err != nil {
	// 	return "", err
	// }

	// if conf.IsAgentTool(command) {
	// 	argm, err := conf.ParseArguments(strings.Join(argv, " "))
	// 	result, err := vars.RootAgent.Runner.Run(ctx, command, argm)
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	return api.ToString(result), nil
	// }
	// return ExecCommand(ctx, r.os, vars, command, argv)

	// argm, err := conf.ParseArguments(strings.Join(argv, " "))
	// if err != nil {
	// 	return "", err
	// }
	// maps.Copy(args, argm)
	result, err := vars.RootAgent.Runner.Run(ctx, command, args)
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
