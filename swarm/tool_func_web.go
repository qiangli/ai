package swarm

import (
	"context"
	"fmt"

	webtool "github.com/qiangli/ai/internal/web/tool"
	"github.com/qiangli/ai/swarm/api"
)

func (r *FuncKit) FetchContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	link, err := GetStrProp("url", args)
	if err != nil {
		return "", err
	}
	return webtool.Fetch(ctx, link)
}

func (r *FuncKit) DownloadContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	link, err := GetStrProp("url", args)
	if err != nil {
		return "", err
	}
	file, err := GetStrProp("file", args)
	if err != nil {
		return "", err
	}
	return webtool.Download(ctx, link, file)
}

// Search the web using DuckDuckGo.
func (r *FuncKit) DdgSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	query, err := GetStrProp("query", args)
	if err != nil {
		return "", err
	}
	max, err := GetIntProp("max_results", args)
	if err != nil {
		return "", err
	}
	if max <= 0 {
		max = 1
	}
	if max > 10 {
		max = 10
	}
	return webtool.DDG(ctx, query, max)
}

// Search the web using Bing.
func (r *FuncKit) BingSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	query, err := GetStrProp("query", args)
	if err != nil {
		return "", err
	}
	max, err := GetIntProp("max_results", args)
	if err != nil {
		return "", err
	}
	if max <= 0 {
		max = 1
	}
	if max > 10 {
		max = 10
	}
	return webtool.Bing(ctx, query, max)
}

// Search the web using Brave.
func (r *FuncKit) BraveSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	query, err := GetStrProp("query", args)
	if err != nil {
		return "", err
	}
	max, err := GetIntProp("max_results", args)
	if err != nil {
		return "", err
	}
	if max <= 0 {
		max = 1
	}
	if max > 10 {
		max = 10
	}

	apiKey, err := varsEnv(vars, "BRAVE_API_KEY")
	if err != nil {
		return "", err
	}
	return webtool.Brave(ctx, apiKey, query, max)
}

// Search the web using Google.
func (r *FuncKit) GoogleSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	query, err := GetStrProp("query", args)
	if err != nil {
		return "", err
	}
	max, err := GetIntProp("max_results", args)
	if err != nil {
		return "", err
	}
	if max <= 0 {
		max = 1
	}
	if max > 10 {
		max = 10
	}

	apiKey, err := varsEnv(vars, "GOOGLE_API_KEY")
	if err != nil {
		return "", err
	}
	seID, err := varsEnv(vars, "GOOGLE_SEARCH_ENGINE_ID")
	if err != nil {
		return "", err
	}
	return webtool.Google(ctx, apiKey, seID, query, max)
}

func varsEnv(vars *api.Vars, key string) (string, error) {
	if vars == nil || vars.Config == nil || vars.Config.Env == nil {
		return "", fmt.Errorf("missing %s", key)
	}
	if v, ok := vars.Config.Env[key]; ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("missing %s", key)
}
