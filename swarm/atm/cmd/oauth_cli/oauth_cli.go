package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
)

const (
	// Base directory for credentials and tokens. Defaults to WORKSPACE env or hardcoded path for compatibility.
	defaultWorkspace = "/Users/liqiang/workspace"
	listenAddr       = ":8080"
	redirectURL      = "http://localhost:8080"
)

var (
	credentialsPath string
	tokensPath      string
)

func init() {
	base := os.Getenv("WORKSPACE")
	if base == "" {
		base = defaultWorkspace
	}
	// Prefer WORKSPACE env, else default to hardcoded workspace path
	credentialsPath = filepath.Join(base, "credentials.json")
	// Compatibility: also look for credentials under opt/oauth_cli if present
	if _, err := os.Stat(credentialsPath); os.IsNotExist(err) {
		alt := filepath.Join(base, "opt", "oauth_cli", "credentials.json")
		if _, err := os.Stat(alt); err == nil {
			credentialsPath = alt
		}
	}

	// Tokens should be stored at workspace root tokens.json with restrictive permissions
	tokensPath = filepath.Join(base, "tokens.json")

}

// credentials.json expected to contain the OAuth client config JSON from Google

func openBrowser(url string) error {
	// Cross-platform opener using exec.Command, no external dependency
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		cmd := exec.Command("cmd", "/c", "start", url)
		return cmd.Start()
	default:
		return exec.Command("xdg-open", url).Start()
	}
}

func saveToken(path string, token *oauth2.Token) error {
	b, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	// write with 0600
	err = ioutil.WriteFile(path, b, 0600)
	if err != nil {
		return err
	}
	return nil
}

func tokenFromFile(path string) (*oauth2.Token, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t oauth2.Token
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func main() {
	ctx := context.Background()

	b, err := ioutil.ReadFile(credentialsPath)
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	conf, err := google.ConfigFromJSON(b, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	conf.RedirectURL = redirectURL

	// Try to load token from file
	tok, err := tokenFromFile(tokensPath)
	if err == nil {
		// Token exists; create client and try to use it (refresh if needed)
		client := conf.Client(ctx, tok)
		srv, err := calendar.New(client)
		if err == nil {
			listEvents(ctx, srv)
			// Save possibly refreshed token
			if err := saveToken(tokensPath, tok); err != nil {
				log.Printf("Warning: failed to save token: %v", err)
			}
			return
		}
		// fallthrough to auth flow if client creation failed
	}

	// Start local server to receive code
	codeCh := make(chan string)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errStr := q.Get("error"); errStr != "" {
			fmt.Fprintf(w, "Error: %s", errStr)
			codeCh <- ""
			return
		}
		code := q.Get("code")
		if code == "" {
			fmt.Fprintf(w, "No code in request")
			return
		}
		fmt.Fprintf(w, "Authorization received. You can close this window.")
		codeCh <- code
	})

	server := &http.Server{Addr: listenAddr}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()

	authURL := conf.AuthCodeURL("state-token", oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	fmt.Printf("Opening browser to: %s\n", authURL)
	if err := openBrowser(authURL); err != nil {
		fmt.Printf("Failed to open browser: %v\n", err)
		fmt.Printf("Please open the following URL in your browser:\n%s\n", authURL)
	}

	// Wait for code or prompt manual paste
	var code string
	select {
	case code = <-codeCh:
		if code == "" {
			log.Fatalf("Authorization failed or was cancelled")
		}
	case <-time.After(120 * time.Second):
		// timeout; prompt user to paste code
		fmt.Print("Enter the authorization code (from the URL 'code' parameter): ")
		if _, err := fmt.Scan(&code); err != nil {
			log.Fatalf("Failed to read code from stdin: %v", err)
		}
	}

	// shutdown server
	_ = server.Shutdown(ctx)

	// Exchange code for token
	tok, err = conf.Exchange(ctx, code)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}

	if err := saveToken(tokensPath, tok); err != nil {
		log.Fatalf("Failed to save token: %v", err)
	}

	client := conf.Client(ctx, tok)
	srv, err := calendar.New(client)
	if err != nil {
		log.Fatalf("Unable to create Calendar client: %v", err)
	}

	listEvents(ctx, srv)
}

func listEvents(ctx context.Context, srv *calendar.Service) {
	now := time.Now().Format(time.RFC3339)
	events, err := srv.Events.List("primary").ShowDeleted(false).
		SingleEvents(true).TimeMin(now).MaxResults(10).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	if len(events.Items) == 0 {
		fmt.Println("No upcoming events found.")
		return
	}

	fmt.Println("Upcoming events:")
	for _, item := range events.Items {
		start := item.Start.DateTime
		if start == "" {
			start = item.Start.Date
		}
		fmt.Printf("%s - %s\n", start, item.Summary)
	}
}
