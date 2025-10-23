package util

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/qiangli/ai/swarm/tool/web"
)

const maxPageSize = 1024 * 1024

// TODO cloud workspace
func Download(ctx context.Context, url, file string) (string, error) {
	out, err := os.Create(file)
	if err != nil {
		return "", err
	}
	defer out.Close()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", web.UserAgent())

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// TODO cloud storage
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%q downloaded successfully. saved locally as %q", url, file), nil
}

func FetchContent(ctx context.Context, url string, start, max int, raw bool) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("Error creating request for URL %q: %v", url, err)
	}
	req.Header.Set("User-Agent", web.UserAgent())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("Error fetching URL %q: %v", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("Error reading response for %q: %v", url, err)
	}

	content := string(body)

	if !raw {
		if v, err := ExtractTextFromHTML(content); err != nil {
			return "", err
		} else {
			content = v
		}
	}
	if start < 0 {
		start = 0
	}
	size := len(content)

	if start >= size {
		return "", fmt.Errorf("invalid start_index: %v. the size of the page is: %v ", start, size)
	}
	end := min(start+max, size)

	return content[start:end], nil
}

func ExtractTextFromHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("error parsing HTML: %v", err)
	}

	text := doc.Text()
	return text, nil
}
