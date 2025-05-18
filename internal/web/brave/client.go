package brave

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/qiangli/ai/internal/web"
)

// https://api-dashboard.search.brave.com/app/documentation/web-search/query#WebSearchAPIQueryParameters
const searchURL = "https://api.search.brave.com/res/v1/web/search?count=%v&q=%s"

var (
	ErrNoGoodResult = errors.New("no good search results found")
	ErrAPIResponse  = errors.New("brave api responded with error")
)

// https://api-dashboard.search.brave.com/app/documentation/web-search/responses
type WebSearchApiResponse struct {
	Type string `json:"type"`

	WebSearch *Search `json:"web"`
}

type Search struct {
	Type    string          `json:"type"`
	Results []*SearchResult `json:"results"`
}

type SearchResult struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`

	ContentType   string   `json:"content_type"`
	ExtraSnippets []string `json:"extra_snippets"`

	//
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	PageAge     string `json:"page_age"`
	Language    string `json:"language"`
}

// Client defines an HTTP client for communicating with brave.
type Client struct {
	maxResults int
	userAgent  string

	apiKey string
}

// Result defines a search query result type.
type Result struct {
	Title string
	Info  string
	Ref   string
}

// New initializes a Client with arguments for setting a max
// results per search query and a value for the user agent header.
func New(apiKey string, maxResults int) *Client {
	if maxResults <= 0 {
		maxResults = 1
	}

	return &Client{
		apiKey:     apiKey,
		maxResults: maxResults,
		userAgent:  web.UserAgent(),
	}
}

func (client *Client) newRequest(ctx context.Context, queryURL string) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating brave request: %w", err)
	}

	if client.userAgent != "" {
		request.Header.Add("User-Agent", client.userAgent)
	}

	return request, nil
}

// Search performs a search query and returns
// the result as string and an error if any.
func (client *Client) Search(ctx context.Context, query string) (string, error) {
	queryURL := fmt.Sprintf(searchURL, client.maxResults, url.QueryEscape(query))

	request, err := client.newRequest(ctx, queryURL)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Encoding", "gzip")
	request.Header.Set("X-Subscription-Token", client.apiKey)

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("get %s error: %w", queryURL, err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", ErrAPIResponse
	}

	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return "", err
		}
		defer gzReader.Close()
		reader = gzReader
	default:
		reader = resp.Body
	}

	body, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	var searchResult WebSearchApiResponse

	if err := json.Unmarshal(body, &searchResult); err != nil {
		return "", err
	}

	if searchResult.WebSearch == nil || len(searchResult.WebSearch.Results) == 0 {
		return "", fmt.Errorf("Empty result")
	}

	var results []*Result

	for _, v := range searchResult.WebSearch.Results {
		results = append(results, &Result{
			Title: v.Title,
			Info:  v.Description,
			Ref:   v.URL,
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
		return "No results were found for your search query. This could be due to brave's bot detection or the query returned no matches. Please try rephrasing your search or try again in a few minutes."
	}

	formattedResults := fmt.Sprintf("Found %d search results:\n\n", len(results))

	for i, result := range results {
		formattedResults += fmt.Sprintf("%d. Title: %s\nDescription: %s\nURL: %s\n\n", (i + 1), result.Title, result.Info, result.Ref)
	}

	return formattedResults
}
