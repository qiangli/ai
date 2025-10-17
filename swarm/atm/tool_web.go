package atm

import (
	"context"
	"math/rand"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	webtool "github.com/qiangli/ai/swarm/tool/web/util"
)

// webAuthKit must be per tool/func call
type webAuthKit struct {
	secrets api.SecretStore
	owner   string
	key     string
}

func (r *webAuthKit) token() (string, error) {
	return r.secrets.Get(r.owner, r.key)
}

func (r *webAuthKit) FetchContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	link, err := GetStrProp("url", args)
	if err != nil {
		return "", err
	}

	log.GetLogger(ctx).Debugf("○ fetching url: %q\n", link)
	content, err := webtool.Fetch(ctx, link)
	log.GetLogger(ctx).Debugf("  content length: %v error: %v\n", len(content), err)
	return content, err
}

func (r *webAuthKit) DownloadContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	link, err := GetStrProp("url", args)
	if err != nil {
		return "", err
	}
	file, err := GetStrProp("file", args)
	if err != nil {
		return "", err
	}

	log.GetLogger(ctx).Debugf("💾 downloading %q to %q \n", link, file)
	return webtool.Download(ctx, link, file)
}

// Search the web using available search tools.
func (r *webAuthKit) Search(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

	ddg := func() (string, error) {
		log.GetLogger(ctx).Debugf("🦆 search ddg query: %q max: %d\n", query, max)
		return webtool.DDG(ctx, query, max)
	}
	bing := func() (string, error) {
		log.GetLogger(ctx).Debugf("🅱️ search bing query: %q max: %d\n", query, max)
		return webtool.Bing(ctx, query, max)
	}
	// brave := func() (string, error) {
	// 	apiKey, err := r.token()
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	log.GetLogger(ctx).Debugf("🦁 brave query: %q max: %d\n", query, max)
	// 	return webtool.Brave(ctx, apiKey, query, max)
	// }
	// google := func() (string, error) {
	// 	// engine_id:api_key
	// 	key, err := r.token()
	// 	if err != nil {
	// 		return "", err
	// 	}
	// 	seID, apiKey := split2(key, ":", "")

	// 	log.GetLogger(ctx).Debugf("🅖 google query: %q max: %d\n", query, max)
	// 	return webtool.Google(ctx, apiKey, seID, query, max)
	// }

	var tools = []func() (string, error){
		bing,
		ddg,
		// brave,
		// google,
	}

	// random
	idx := rand.Intn(len(tools))
	return tools[idx]()
}

// Search the web using DuckDuckGo.
func (r *webAuthKit) DdgSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

	log.GetLogger(ctx).Debugf("🦆 ddg query: %q max: %d\n", query, max)
	return webtool.DDG(ctx, query, max)
}

// Search the web using Bing.
func (r *webAuthKit) BingSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

	log.GetLogger(ctx).Debugf("🅱️ bing query: %q max: %d\n", query, max)
	return webtool.Bing(ctx, query, max)
}

// Search the web using Brave.
func (r *webAuthKit) BraveSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

	apiKey, err := r.token()
	if err != nil {
		return "", err
	}

	log.GetLogger(ctx).Debugf("🦁 brave query: %q max: %d\n", query, max)
	return webtool.Brave(ctx, apiKey, query, max)
}

// Search the web using Google.
func (r *webAuthKit) GoogleSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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
	key, err := r.token()
	if err != nil {
		return "", err
	}
	seID, apiKey := split2(key, ":", "")

	log.GetLogger(ctx).Debugf("🅖 google query: %q max: %d\n", query, max)
	return webtool.Google(ctx, apiKey, seID, query, max)
}

// wrapper for webAuthKit
type WebKit struct {
	secrets api.SecretStore
}

func NewWebKit(secrets api.SecretStore) *WebKit {
	return &WebKit{
		secrets: secrets,
	}
}

func (r *WebKit) Call(ctx context.Context, vars *api.Vars, env *api.ToolEnv, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}

	// forward to web auth kit
	wk := &webAuthKit{
		secrets: r.secrets,
		owner:   env.Owner,
		key:     tf.ApiKey,
	}
	return CallKit(wk, tf.Kit, tf.Name, callArgs...)
}
