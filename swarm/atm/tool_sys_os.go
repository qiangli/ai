package atm

import (
	"context"

	"github.com/qiangli/ai/swarm/api"
)

// func (r *SystemKit) Man(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	command, err := api.GetStrProp("command", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return r.os.Man(command)
// }

func (r *SystemKit) Run(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	command, err := api.GetStrProp("command", args)
	if err != nil {
		return "", err
	}
	argsList, err := api.GetArrayProp("args", args)
	if err != nil {
		return "", err
	}
	return RunRestricted(ctx, r.user, r.secrets, r.os, vars, command, argsList)
}

func (r *SystemKit) Cd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return "", err
	}
	return "", r.os.Chdir(dir)
}

func (r *SystemKit) Pwd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return r.os.Getwd()
}
