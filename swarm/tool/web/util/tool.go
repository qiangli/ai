package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/qiangli/ai/swarm/tool/web/bing"
	"github.com/qiangli/ai/swarm/tool/web/brave"
	"github.com/qiangli/ai/swarm/tool/web/ddg"
	"github.com/qiangli/ai/swarm/tool/web/google"
	"github.com/qiangli/ai/swarm/tool/web/scrape"
)

const maxPageSize = 8000

func Download(ctx context.Context, url, file string) (string, error) {
	out, err := os.Create(file)
	if err != nil {
		return "", err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%q downloaded succesfully. saved locally as %q", url, file), nil
}

// Fetch and parse content from a webpage
func Fetch(ctx context.Context, url string) (string, error) {
	scraper, err := scrape.New()
	if err != nil {
		return "", err
	}
	result, err := scraper.Fetch(ctx, url)
	if len(result) > maxPageSize {
		result = result[:maxPageSize]
	}
	return result, err
}

// Search performs a search query and returns the result as string and an error if any.
func DDG(ctx context.Context, query string, maxResults int) (string, error) {
	cli := ddg.New(maxResults)
	return cli.Search(ctx, query)
}

func Bing(ctx context.Context, query string, maxResults int) (string, error) {
	cli := bing.New(maxResults)
	return cli.Search(ctx, query)
}

func Brave(ctx context.Context, apiKey, query string, maxResults int) (string, error) {
	cli := brave.New(apiKey, maxResults)
	return cli.Search(ctx, query)
}

func Google(ctx context.Context, apiKey, searchEngineID, query string, maxResults int) (string, error) {
	cli := google.New(apiKey, searchEngineID, maxResults)
	return cli.Search(ctx, query)
}
