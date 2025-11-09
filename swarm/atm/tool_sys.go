package atm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/shell/tool/sh/vfs"
	"github.com/qiangli/shell/tool/sh/vos"
)

type SystemKit struct {
	workspace string
	user      *api.User
	fs        vfs.FileSystem
	os        vos.System
	secrets   api.SecretStore
}

func NewSystemKit(workspace string, user *api.User, fs vfs.FileSystem, os vos.System, secrets api.SecretStore) *SystemKit {
	return &SystemKit{
		workspace: workspace,
		user:      user,
		fs:        fs,
		os:        os,
		secrets:   secrets,
	}
}

func (r *SystemKit) getStr(key string, args map[string]any) (string, error) {
	return GetStrProp(key, args)
}

func (r *SystemKit) getArray(key string, args map[string]any) ([]string, error) {
	return GetArrayProp(key, args)
}

func (r *SystemKit) Call(ctx context.Context, vars *api.Vars, _ *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	v, err := CallKit(r, tf.Kit, tf.Name, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to call system tool %s %s: %w", tf.Kit, tf.Name, err)
	}
	return v, err
}
