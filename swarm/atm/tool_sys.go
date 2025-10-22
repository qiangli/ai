package atm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/vfs"
	"github.com/qiangli/ai/swarm/vos"
)

type SystemKit struct {
	user    *api.User
	fs      vfs.FileSystem
	os      vos.System
	secrets api.SecretStore
}

func NewSystemKit(user *api.User, fs vfs.FileSystem, os vos.System, secrets api.SecretStore) *SystemKit {
	return &SystemKit{
		user:    user,
		fs:      fs,
		os:      os,
		secrets: secrets,
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
	v, err := CallKit(r, tf.Config.Kit, tf.Name, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to call system tool %s %s: %w", tf.Config.Kit, tf.Name, err)
	}
	return v, err

	// var result api.Result
	// if s, ok := v.(string); ok {
	// 	result.Value = s
	// } else if c, ok := v.(*api.Blob); ok {
	// 	result.Content = c.Content
	// 	result.MimeType = c.MimeType
	// } else {
	// 	result.Value = fmt.Sprintf("%v", v)
	// }
	// return &result, nil
}
