package atm

import (
	"context"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
	"github.com/qiangli/ai/swarm/tool/web/scrape"
	webtool "github.com/qiangli/ai/swarm/tool/web/util"
)

var (
	NoResult = "Empty content. This could be due to website bot detection. Please try a different website or try again in a few minutes."
)

// WebKit must be per tool/func call
// type WebKit struct {
// 	// vars *api.Vars
// 	// key  string
// }

func (r *WebKit) token(vars *api.Vars, args map[string]any) (string, error) {
	key, err := api.GetStrProp("api_key", args)
	if err != nil {
		return "", fmt.Errorf("api_key missing")
	}
	return vars.Token(key)
}

func (r *WebKit) FetchContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	link, err := api.GetStrProp("url", args)
	if err != nil {
		return "", err
	}
	start, _ := api.GetIntProp("start_index", args)
	if start < 0 {
		start = 0
	}
	max, _ := api.GetIntProp("max_length", args)
	if max <= 0 {
		max = 8000
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

	content, err := cli.Fetch(ctx, link)
	if err != nil {
		return "", err
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

func (r *WebKit) DownloadContent(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

// Search the web using DuckDuckGo.
func (r *WebKit) DdgSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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
func (r *WebKit) BingSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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
func (r *WebKit) BraveSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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

	token, err := r.token(vars, args)
	if err != nil {
		return "", err
	}

	log.GetLogger(ctx).Debugf("🦁 brave query: %q max: %d\n", query, max)
	return webtool.Brave(ctx, token, query, max)
}

// Search the web using Google.
func (r *WebKit) GoogleSearch(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
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
	token, err := r.token(vars, args)
	if err != nil {
		return "", err
	}
	seID, apiKey := split2(token, ":", "")

	log.GetLogger(ctx).Debugf("🅖 google query: %q max: %d\n", query, max)
	return webtool.Google(ctx, apiKey, seID, query, max)
}

type WebKit struct {
}

func NewWebKit() *WebKit {
	return &WebKit{}
}

func (r *WebKit) Call(ctx context.Context, vars *api.Vars, _ *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, tf.Name, args}

	// // forward to web auth kit
	// ak := &WebKit{
	// 	// vars: vars,
	// 	// key:  tf.ApiKey,
	// }

	// mock if echo__<id> is found in args.
	if len(args) > 0 {
		if v, found := args["adapter"]; found && api.ToString(v) == "echo" {
			return echoAdapter(args)
		}
	}

	result, err := CallKit(r, tf.Kit, tf.Name, callArgs...)

	if err != nil {
		return nil, fmt.Errorf("error: %v. please try again after few seconds.", err)
	}
	return result, nil
}
