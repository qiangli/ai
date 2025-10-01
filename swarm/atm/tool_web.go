package atm

import (
	"context"

	// log "github.com/sirupsen/logrus"

	"github.com/qiangli/ai/swarm/api"
	swarmlog "github.com/qiangli/ai/swarm/log"
	webtool "github.com/qiangli/ai/swarm/tool/web/util"
)

// WebKit must be per tool/func call
type WebKit struct {
	apiKey func() (string, error)
}

func (r *WebKit) FetchContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	link, err := GetStrProp("url", args)
	if err != nil {
		return "", err
	}

	swarmlog.GetLogger(ctx).Debugf("○ fetching url: %q\n", link)
	return webtool.Fetch(ctx, link)
}

func (r *WebKit) DownloadContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	link, err := GetStrProp("url", args)
	if err != nil {
		return "", err
	}
	file, err := GetStrProp("file", args)
	if err != nil {
		return "", err
	}

	swarmlog.GetLogger(ctx).Debugf("💾 downloading %q to %q \n", link, file)
	return webtool.Download(ctx, link, file)
}

// Search the web using DuckDuckGo.
func (r *WebKit) DdgSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

	swarmlog.GetLogger(ctx).Debugf("🦆 ddg query: %q max: %d\n", query, max)
	return webtool.DDG(ctx, query, max)
}

// Search the web using Bing.
func (r *WebKit) BingSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

	swarmlog.GetLogger(ctx).Debugf("🅱️ bing query: %q max: %d\n", query, max)
	return webtool.Bing(ctx, query, max)
}

// Search the web using Brave.
func (r *WebKit) BraveSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

	apiKey, err := r.apiKey()
	if err != nil {
		return "", err
	}

	swarmlog.GetLogger(ctx).Debugf("🦁 brave query: %q max: %d\n", query, max)
	return webtool.Brave(ctx, apiKey, query, max)
}

// Search the web using Google.
func (r *WebKit) GoogleSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

	// engine_id:api_key
	key, err := r.apiKey()
	if err != nil {
		return "", err
	}
	seID, apiKey := split2(key, ":", "")

	swarmlog.GetLogger(ctx).Debugf("🅖 google query: %q max: %d\n", query, max)
	return webtool.Google(ctx, apiKey, seID, query, max)
}

func (r *WebKit) callTool(ctx context.Context, vars *api.Vars, f *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, f.Name, args}

	return CallKit(r, f.Kit, f.Name, callArgs...)
}
