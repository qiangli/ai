package atm

import (
	"context"
	"fmt"
	"time"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/tool/web/scrape"
	webtool "github.com/qiangli/ai/swarm/tool/web/util"
)

var (
	NoResult = "Empty content. This could be due to website bot detection. Please try a different website or try again in a few minutes."
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
	link, err := api.GetStrProp("url", args)
	if err != nil {
		return "", err
	}
	start, err := api.GetIntProp("start_index", args)
	if err != nil {
		return "", err
	}
	max, err := api.GetIntProp("max_length", args)
	if err != nil {
		return "", err
	}
	// raw, err := GetBoolProp("raw", args)
	// if err != nil {
	// 	return "", err
	// }

	log.GetLogger(ctx).Debugf("○ fetching url: %q\n", link)
	// content, err := webtool.FetchContent(ctx, link, start, max, raw)
	cli, err := scrape.New()
	if err != nil {
		return "", err
	}

	var content string
	for range 3 {
		if v, err := cli.Fetch(ctx, link); err != nil || len(v) == 0 {
			time.Sleep(1 * time.Second)
			continue
		} else {
			content = v
			break
		}
	}

	size := len(content)
	if size == 0 {
		return NoResult, nil
	}
	if start < 0 {
		start = 0
	}
	if start >= size {
		return "", fmt.Errorf("invalid start_index: %v. the size of the page is: %v ", start, size)
	}
	end := min(start+max, size)
	content = content[start:end]

	log.GetLogger(ctx).Debugf("  content length: %v error: %v\n", len(content), err)
	return content, err
}

func (r *webAuthKit) DownloadContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	link, err := api.GetStrProp("url", args)
	if err != nil {
		return "", err
	}
	file, err := api.GetStrProp("file", args)
	if err != nil {
		return "", err
	}

	log.GetLogger(ctx).Debugf("💾 downloading %q to %q \n", link, file)
	return webtool.Download(ctx, link, file)
}

// this should be done with an agent in a much more flexible way.
// // Search the web using available search tools.
// func (r *webAuthKit) Search(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
// 	query, err := api.GetStrProp("query", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	max, err := api.GetIntProp("max_results", args)
// 	if err != nil {
// 		return "", err
// 	}
// 	if max <= 0 {
// 		max = 1
// 	}
// 	if max > 10 {
// 		max = 10
// 	}

// 	ddg := func() (string, error) {
// 		log.GetLogger(ctx).Debugf("🦆 search ddg query: %q max: %d\n", query, max)
// 		return webtool.DDG(ctx, query, max)
// 	}
// 	bing := func() (string, error) {
// 		log.GetLogger(ctx).Debugf("🅱️ search bing query: %q max: %d\n", query, max)
// 		return webtool.Bing(ctx, query, max)
// 	}
// 	// brave := func() (string, error) {
// 	// 	apiKey, err := r.token()
// 	// 	if err != nil {
// 	// 		return "", err
// 	// 	}
// 	// 	log.GetLogger(ctx).Debugf("🦁 brave query: %q max: %d\n", query, max)
// 	// 	return webtool.Brave(ctx, apiKey, query, max)
// 	// }
// 	// google := func() (string, error) {
// 	// 	// engine_id:api_key
// 	// 	key, err := r.token()
// 	// 	if err != nil {
// 	// 		return "", err
// 	// 	}
// 	// 	seID, apiKey := split2(key, ":", "")

// 	// 	log.GetLogger(ctx).Debugf("🅖 google query: %q max: %d\n", query, max)
// 	// 	return webtool.Google(ctx, apiKey, seID, query, max)
// 	// }

// 	var tools = []func() (string, error){
// 		bing,
// 		ddg,
// 		// brave,
// 		// google,
// 	}

// 	// random
// 	idx := rand.Intn(len(tools))
// 	return tools[idx]()
// }

// Search the web using DuckDuckGo.
func (r *webAuthKit) DdgSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	query, err := api.GetStrProp("query", args)
	if err != nil {
		return "", err
	}
	max, err := api.GetIntProp("max_results", args)
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
	query, err := api.GetStrProp("query", args)
	if err != nil {
		return "", err
	}
	max, err := api.GetIntProp("max_results", args)
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
	query, err := api.GetStrProp("query", args)
	if err != nil {
		return "", err
	}
	max, err := api.GetIntProp("max_results", args)
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
	query, err := api.GetStrProp("query", args)
	if err != nil {
		return "", err
	}
	max, err := api.GetIntProp("max_results", args)
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
	result, err := CallKit(wk, tf.Kit, tf.Name, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("error: %v. please try again after few seconds.", err)
	}
	return result, nil
}
