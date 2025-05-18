package swarm

import (
	"context"
	"os"

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
	// TODO move to app confg?
	apiKey := os.Getenv("BRAVE_API_KEY")
	return webtool.Brave(ctx, apiKey, query, max)
}
