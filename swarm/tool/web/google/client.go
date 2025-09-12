package google

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"

	"github.com/qiangli/ai/swarm/tool/web"
)

func (client *Client) customSearch(ctx context.Context, query string) (*customsearch.Search, error) {
	var options []option.ClientOption
	options = append(options, option.WithAPIKey(client.apiKey))
	if client.userAgent != "" {
		options = append(options, option.WithUserAgent(client.userAgent))
	}
	svc, err := customsearch.NewService(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("Unable to create customsearch service: %v", err)
	}
	resp, err := svc.Cse.List().Cx(client.searchEngineID).Q(query).Num(int64(client.maxResults)).Do()
	if err != nil {
		return nil, ErrAPIResponse
	}

	return resp, nil
}

var (
	NoResult = "No results were found for your search query. This could be due to Google's bot detection or the query returned no matches. Please try rephrasing your search or try again in a few minutes."

	ErrAPIResponse = errors.New("google api responded with error")
)

// Client defines an HTTP client for communicating with google.
type Client struct {
	apiKey         string
	searchEngineID string

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
func New(apiKey, searchEngineID string, maxResults int) *Client {
	if maxResults <= 0 {
		maxResults = 1
	}

	return &Client{
		apiKey:         apiKey,
		searchEngineID: searchEngineID,
		maxResults:     maxResults,
		userAgent:      web.UserAgent(),
	}
}

// Search performs a search query and returns
// the result as string and an error if any.
func (client *Client) Search(ctx context.Context, query string) (string, error) {
	var results []*Result

	resp, err := client.customSearch(ctx, query)
	if err != nil {
		return "", err
	}
	for _, v := range resp.Items {
		results = append(results, &Result{
			Title: v.Title,
			Info:  v.Snippet,
			Ref:   v.Link,
		})
	}

	return client.formatResults(results), nil
}

func (client *Client) SetMaxResults(n int) {
	client.maxResults = n
}

// formatResults will return a structured string with the results.
func (client *Client) formatResults(results []*Result) string {
	if len(results) == 0 {
		return NoResult
	}

	formattedResults := fmt.Sprintf("Found %d search results:\n\n", len(results))

	for i, result := range results {
		formattedResults += fmt.Sprintf("%d. Title: %s\nDescription: %s\nURL: %s\n\n", (i + 1), result.Title, result.Info, result.Ref)
	}

	return formattedResults
}
