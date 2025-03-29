// https://github.com/tmc/langchaingo/blob/main/tools/duckduckgo/internal/client.go
// https://github.com/tmc/langchaingo?tab=MIT-1-ov-file#readme
package ddg

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/qiangli/ai/internal/web"
)

// const SafariUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36"
// const EdgeUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36 Edg/119.0.0.0"

var (
	ErrNoGoodResult = errors.New("no good search results found")
	ErrAPIResponse  = errors.New("duckduckgo api responded with error")
)

// Client defines an HTTP client for communicating with duckduckgo.
type Client struct {
	maxResults int
	userAgent  string
}

// Result defines a search query result type.
type Result struct {
	Title string
	Info  string
	Ref   string
}

// New initializes a Client with arguments for setting a max
// results per search query and a value for the user agent header.
func New(maxResults int) *Client {
	if maxResults <= 0 {
		maxResults = 1
	}

	return &Client{
		maxResults: maxResults,
		userAgent:  web.UserAgent(),
	}
}

func (client *Client) newRequest(ctx context.Context, queryURL string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating duckduckgo request: %w", err)
	}

	if client.userAgent != "" {
		request.Header.Add("User-Agent", client.userAgent)
	}

	return request, nil
}

// Search performs a search query and returns
// the result as string and an error if any.
func (client *Client) Search(ctx context.Context, query string) (string, error) {
	queryURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	request, err := client.newRequest(ctx, queryURL)
	if err != nil {
		return "", err
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("get %s error: %w", queryURL, err)
	}

	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", ErrAPIResponse
	}

	doc, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return "", fmt.Errorf("new document error: %w", err)
	}

	results := []Result{}
	sel := doc.Find(".web-result")

	for i := range sel.Nodes {
		// Break loop once required amount of results are add
		if client.maxResults == len(results) {
			break
		}
		node := sel.Eq(i)
		linkNode := node.Find(".result__a")
		title := linkNode.Text()
		ref := linkNode.AttrOr("href", "")

		if title == "" || ref == "" {
			continue
		}
		parts := strings.SplitN(ref, "uddg=", 2)
		if len(parts) > 1 {
			ref, err = url.QueryUnescape(parts[1])
		} else {
			ref, err = url.QueryUnescape(ref)
		}
		if err != nil {
			continue
		}

		info := node.Find(".result__snippet").Text()
		results = append(results, Result{title, info, ref})
	}

	return client.formatResults(results), nil
}

func (client *Client) SetMaxResults(n int) {
	client.maxResults = n
}

// formatResults will return a structured string with the results.
func (client *Client) formatResults(results []Result) string {
	if len(results) == 0 {
		return "No results were found for your search query. This could be due to DuckDuckGo's bot detection or the query returned no matches. Please try rephrasing your search or try again in a few minutes."
	}

	formattedResults := fmt.Sprintf("Found %d search results:\n\n", len(results))

	for i, result := range results {
		formattedResults += fmt.Sprintf("%d. Title: %s\nDescription: %s\nURL: %s\n\n", (i + 1), result.Title, result.Info, result.Ref)
	}

	return formattedResults
}
