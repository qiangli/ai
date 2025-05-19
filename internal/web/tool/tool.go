package tool

import (
	"context"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/web/bing"
	"github.com/qiangli/ai/internal/web/brave"
	"github.com/qiangli/ai/internal/web/ddg"
	"github.com/qiangli/ai/internal/web/google"
	"github.com/qiangli/ai/internal/web/scrape"
)

// Fetch and parse content from a webpage
func Fetch(ctx context.Context, url string) (string, error) {
	log.Infof("🌐 fetching url: %q\n", url)

	scraper, err := scrape.New()
	if err != nil {
		return "", err
	}
	result, err := scraper.Fetch(ctx, url)
	if len(result) > 8000 {
		result = result[:8000]
	}
	return result, err
}

// Search performs a search query and returns the result as string and an error if any.
func DDG(ctx context.Context, query string, maxResults int) (string, error) {
	log.Infof("🦆 ddg query: %q max: %d\n", query, maxResults)

	cli := ddg.New(maxResults)
	return cli.Search(ctx, query)
}

func Bing(ctx context.Context, query string, maxResults int) (string, error) {
	log.Infof("🅱️ bing query: %q max: %d\n", query, maxResults)

	cli := bing.New(maxResults)
	return cli.Search(ctx, query)
}

func Brave(ctx context.Context, apiKey, query string, maxResults int) (string, error) {
	log.Infof("🦁 brave query: %q max: %d\n", query, maxResults)

	cli := brave.New(apiKey, maxResults)
	return cli.Search(ctx, query)
}

func Google(ctx context.Context, apiKey, searchEngineID, query string, maxResults int) (string, error) {
	log.Infof("🅖 google query: %q max: %d\n", query, maxResults)

	cli := google.New(apiKey, searchEngineID, maxResults)
	return cli.Search(ctx, query)
}
