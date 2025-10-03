package atm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
)

type SystemKit struct {
}

func NewSystemKit() *SystemKit {
	return &SystemKit{}
}

func (r *SystemKit) getStr(key string, args map[string]any) (string, error) {
	return GetStrProp(key, args)
}

func (r *SystemKit) getArray(key string, args map[string]any) ([]string, error) {
	return GetArrayProp(key, args)
}

func (r *SystemKit) Call(ctx context.Context, vars *api.Vars, token api.SecretToken, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}
	v, err := CallKit(r, tf.Config.Kit, tf.Name, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to call system tool %s %s: %w", tf.Config.Kit, tf.Name, err)
	}

	var result api.Result
	if s, ok := v.(string); ok {
		result.Value = s
	} else if c, ok := v.(*FileContent); ok {
		result.Value = string(c.Content)
		result.MimeType = c.MimeType
		result.Message = c.Message
	} else {
		result.Value = fmt.Sprintf("%v", v)
	}
	return &result, nil
}
