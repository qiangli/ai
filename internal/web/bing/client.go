package bing

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"

	"github.com/qiangli/ai/internal/web"
)

const searchURL = "https://www.bing.com/search?q=%s"

var (
	ErrNoGoodResult = errors.New("no good search results found")
	ErrAPIResponse  = errors.New("bing responded with error")
)

type Client struct {
	maxResults int
	userAgent  string
}

type Result struct {
	Title string
	Info  string
	Ref   string
}

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
		return nil, fmt.Errorf("creating bing request: %w", err)
	}

	if client.userAgent != "" {
		request.Header.Add("User-Agent", client.userAgent)
	}

	return request, nil
}

func (client *Client) Search(ctx context.Context, query string) (string, error) {
	queryURL := fmt.Sprintf(searchURL, url.QueryEscape(query))

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

	var results []*Result

	sel := doc.Find(".b_algo")

	for i := range sel.Nodes {
		// Break loop once required amount of results are add
		if client.maxResults == len(results) {
			break
		}
		node := sel.Eq(i)

		h2Node := node.Find("h2")
		linkNode := h2Node.Find("a")
		title := linkNode.Text()
		ref := linkNode.AttrOr("href", "")

		info := node.Find("p").Text()
		if title == "" || ref == "" {
			continue
		}
		results = append(results, &Result{title, info, ref})
	}

	return client.formatResults(results), nil
}

func (client *Client) SetMaxResults(n int) {
	client.maxResults = n
}

func (client *Client) formatResults(results []*Result) string {
	if len(results) == 0 {
		return "No results were found for your search query. This could be due to bing's bot detection or the query returned no matches. Please try rephrasing your search or try again in a few minutes."
	}

	formattedResults := fmt.Sprintf("Found %d search results:\n\n", len(results))

	for i, result := range results {
		formattedResults += fmt.Sprintf("%d. Title: %s\nDescription: %s\nURL: %s\n\n", (i + 1), result.Title, result.Info, result.Ref)
	}

	return formattedResults
}
