package lang

import (
	"context"
	"testing"
)

func TestGolang(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		script    string
		want      string
		expectErr bool
	}{
		{"r := 1 + 1;print(r)", "2", false},
		{`
package main

import "fmt"

func test() string {
	fmt.Println("hello")
	return "ok"
}
func main() {
	test()
}
		`, "hello\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.script, func(t *testing.T) {
			got, err := Golang(ctx, nil, nil, tt.script, nil)
			if (err != nil) != tt.expectErr {
				t.Errorf("Go() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if got != tt.want {
				t.Errorf("Go() = %v, want %v", got, tt.want)
			}
			t.Logf("got: %s", got)
		})
	}
}

func TestGolangFetch(t *testing.T) {
	ctx := context.Background()
	var script = `
package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

func fetchWebPage(url string) {
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error fetching the webpage:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error: received non-200 response code: %d\n", resp.StatusCode)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	fmt.Println(string(body))
}

func main() {
	url := "https://example.com"
	fetchWebPage(url)
}
`
	got, err := Golang(ctx, nil, nil, script, nil)
	if err != nil {
		t.FailNow()
	}
	t.Logf("got: %v", got)
}
