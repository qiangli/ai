package swarm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	webtool "github.com/qiangli/ai/swarm/tool/web/util"
)

type WebKit struct {
	apiKey func() (string, error)
}

func (r *WebKit) FetchContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	link, err := GetStrProp("url", args)
	if err != nil {
		return "", err
	}

	log.GetLogger(ctx).Info("○ fetching url: %q\n", link)
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

	log.GetLogger(ctx).Info("💾 downloading %q to %q \n", link, file)
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

	log.GetLogger(ctx).Info("🦆 ddg query: %q max: %d\n", query, max)
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

	log.GetLogger(ctx).Info("🅱️ bing query: %q max: %d\n", query, max)
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

	log.GetLogger(ctx).Info("🦁 brave query: %q max: %d\n", query, max)
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

	// TODO handel engine_id
	key, err := r.apiKey()
	if err != nil {
		return "", err
	}

	seID, apiKey := split2(key, ":", "")
	// // seID, err := r.env("GOOGLE_SEARCH_ENGINE_ID")
	// if r.f.Extra == nil {
	// 	return "", fmt.Errorf("missing search engine id")
	// }
	// seID, ok := r.f.Extra["engine_id"]
	// if !ok || len(seID) == 0 {
	// 	return "", fmt.Errorf("empty search engine id")
	// }

	log.GetLogger(ctx).Info("🅖 google query: %q max: %d\n", query, max)
	return webtool.Google(ctx, apiKey, seID, query, max)
}

func callWebTool(ctx context.Context, vars *api.Vars, f *api.ToolFunc, args map[string]any) (string, error) {
	tool := &WebKit{
		apiKey: provideApiKey(f.ApiKey),
	}
	callArgs := []any{ctx, vars, f.Name, args}
	v, err := CallKit(tool, f.Kit, f.Name, callArgs...)
	if err != nil {
		return "", fmt.Errorf("failed to call function tool %s %s: %w", f.Kit, f.Name, err)
	}
	if s, ok := v.(string); ok {
		return s, nil
	}

	return fmt.Sprintf("%v", v), nil
}
