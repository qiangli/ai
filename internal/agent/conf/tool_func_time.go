package conf

import (
	"context"
	"strings"
	"time"

	"github.com/qiangli/ai/swarm/api"
	utool "github.com/qiangli/ai/swarm/tool/util"
)

func (r *FuncKit) GetLocalTimezone(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	tz, err := utool.GetLocalTZ("")
	if err != nil {
		return "", err
	}
	// FIXME
	tzName := tz.String()
	if strings.ToLower(tzName) == "local" {
		ct := time.Now().In(tz)
		tzName = ct.Format("MST")
	}
	return tzName, nil
}

func (r *FuncKit) GetCurrentTime(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	tz, err := GetStrProp("timezone", args)
	if err != nil {
		return "", err
	}
	result, err := utool.GetCurrentTime(tz)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func (r *FuncKit) ConvertTime(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	source, err := GetStrProp("source_timezone", args)
	if err != nil {
		return "", err
	}
	target, err := GetStrProp("target_timezone", args)
	if err != nil {
		return "", err
	}
	t, err := GetStrProp("time", args)
	if err != nil {
		return "", err
	}

	result, err := utool.ConvertTime(source, t, target)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}
