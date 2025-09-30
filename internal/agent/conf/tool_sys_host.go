package conf

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"strings"

	"github.com/qiangli/ai/swarm/api"
	utool "github.com/qiangli/ai/swarm/tool/util"
)

func (r *SystemKit) getStr(key string, args map[string]any) (string, error) {
	return GetStrProp(key, args)
}

func (r *SystemKit) getArray(key string, args map[string]any) ([]string, error) {
	return GetArrayProp(key, args)
}

func (r *SystemKit) ListCommands(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	list := _os.ListCommands()
	return strings.Join(list, "\n"), nil
}

func (r *SystemKit) Which(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	commands, err := r.getArray("commands", args)
	if err != nil {
		return "", err
	}
	return _os.Which(commands)
}

// func (r *SystemKit) Man(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	command, err := r.getStr("command", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return _os.Man(command)
// }

// func (r *SystemKit) Exec(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	command, err := r.getStr("command", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	argsList, err := r.getArray("args", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return runRestricted(ctx, vars, r.agent, command, argsList)
// }

// func (r *SystemKit) Cd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	dir, err := r.getStr("dir", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	return "", _os.Chdir(dir)
// }

// func (r *SystemKit) Pwd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	return _os.Getwd()
// }

func (r *SystemKit) Env(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return _os.Env(), nil
}

func (r *SystemKit) Uname(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	os, arch := _os.Uname()
	return fmt.Sprintf("OS: %s\nArch: %s", os, arch), nil
}

func (r *SystemKit) WhoAmI(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return utool.WhoAmI()
}

func (r *SystemKit) HomeDir(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.Home, nil
}
func (r *SystemKit) TempDir(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.Temp, nil
}
func (r *SystemKit) WorkspaceDir(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.Workspace, nil
}
func (r *SystemKit) RepoDir(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.Workspace, nil
	// return vars.Repo, nil
}
