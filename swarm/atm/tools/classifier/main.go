package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type envelope struct {
	ID      string          `json:"id,omitempty"`
	User    string          `json:"user,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type input struct {
	Query string `json:"query"`
}

type output struct {
	ID         string   `json:"id,omitempty"`
	User       string   `json:"user,omitempty"`
	Action     string   `json:"action"`
	Dir        string   `json:"dir,omitempty"`
	Args       []string `json:"args,omitempty"`
	Message    string   `json:"message,omitempty"`
	Confidence float64  `json:"confidence"`
	Error      string   `json:"error,omitempty"`
}

var (
	reDir1 = regexp.MustCompile(`(?i)\b(in|within|under)\s+([^\s]+)`) // crude
	reDir2 = regexp.MustCompile(`(?i)\bdir\s*[:=]\s*([^\s]+)`)
)

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		write(output{Confidence: 0, Error: fmt.Sprintf("read stdin: %v", err)})
		os.Exit(2)
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		write(output{Confidence: 0, Error: "empty stdin; expected JSON"})
		os.Exit(2)
	}

	var env envelope
	var in input
	// Accept either envelope or bare input.
	if err := json.Unmarshal(raw, &env); err == nil && len(env.Payload) > 0 {
		if err := json.Unmarshal(env.Payload, &in); err != nil {
			write(output{ID: env.ID, User: env.User, Confidence: 0, Error: fmt.Sprintf("decode payload: %v", err)})
			os.Exit(2)
		}
	} else {
		if err := json.Unmarshal(raw, &in); err != nil {
			write(output{Confidence: 0, Error: fmt.Sprintf("decode input: %v", err)})
			os.Exit(2)
		}
	}

	res := classify(in.Query)
	res.ID = env.ID
	res.User = env.User
	write(res)
	if res.Error != "" {
		os.Exit(2)
	}
}

func classify(q string) output {
	q = strings.TrimSpace(q)
	if q == "" {
		return output{Confidence: 0, Error: "query must not be empty"}
	}
	lq := strings.ToLower(q)

	dir := extractDir(q)

	// Order matters: more specific first.
	switch {
	case strings.Contains(lq, "latest commit") || strings.Contains(lq, "last commit"):
		return output{Action: "latest-commit", Dir: dir, Confidence: 1.0}
	case strings.Contains(lq, "list branches") || strings.Contains(lq, "branches") || strings.Contains(lq, "list-branches"):
		return output{Action: "list-branches", Dir: dir, Confidence: 1.0}
	case strings.Contains(lq, "list remotes") || strings.Contains(lq, "remotes") || strings.Contains(lq, "list-remotes"):
		return output{Action: "list-remotes", Dir: dir, Confidence: 1.0}
	case strings.Contains(lq, "remote url") || strings.Contains(lq, "remote-url"):
		return output{Action: "remote-url", Dir: dir, Confidence: 1.0}
	case strings.Contains(lq, "rev-parse"):
		rev := lastToken(q)
		conf := 0.5
		if rev != "" && rev != "rev-parse" {
			conf = 1.0
		}
		return output{Action: "rev-parse", Dir: dir, Args: nonEmpty([]string{rev}), Confidence: conf}
	case strings.Contains(lq, "show") && strings.Contains(lq, "file"):
		// Expect something like: show file <path> at <rev>
		// Heuristic: take last token as rev and first path-like token as path.
		path := firstPathLikeToken(q)
		rev := lastToken(q)
		conf := 0.5
		if path != "" && rev != "" {
			conf = 1.0
		}
		return output{Action: "show-file", Dir: dir, Args: nonEmpty([]string{rev, path}), Confidence: conf}
	case strings.Contains(lq, "status"):
		return output{Action: "status", Dir: dir, Confidence: 1.0}
	case strings.Contains(lq, "clone"):
		// Needs repo and dest; too hard without a real parser.
		return output{Action: "clone", Dir: dir, Confidence: 0.5}
	case strings.Contains(lq, "commit"):
		// Try to extract message after -m or quotes; else ambiguous.
		msg := extractCommitMessage(q)
		conf := 0.5
		if msg != "" {
			conf = 1.0
		}
		return output{Action: "commit", Dir: dir, Message: msg, Confidence: conf}
	case strings.Contains(lq, "pull"):
		return output{Action: "pull", Dir: dir, Confidence: 1.0}
	case strings.Contains(lq, "push"):
		return output{Action: "push", Dir: dir, Confidence: 1.0}
	case strings.Contains(lq, "current branch") || (strings.Contains(lq, "branch") && !strings.Contains(lq, "list branches")):
		return output{Action: "branch", Dir: dir, Confidence: 1.0}
	default:
		return output{Action: "", Dir: dir, Confidence: 0.0}
	}
}

func extractDir(q string) string {
	if m := reDir2.FindStringSubmatch(q); len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	if m := reDir1.FindStringSubmatch(q); len(m) == 3 {
		return strings.TrimSpace(m[2])
	}
	return ""
}

func lastToken(q string) string {
	fields := strings.Fields(q)
	if len(fields) == 0 {
		return ""
	}
	return strings.Trim(fields[len(fields)-1], "\"'")
}

func firstPathLikeToken(q string) string {
	for _, tok := range strings.Fields(q) {
		clean := strings.Trim(tok, "\"'")
		if strings.Contains(clean, "/") || strings.Contains(clean, ".") {
			return clean
		}
	}
	return ""
}

func extractCommitMessage(q string) string {
	lq := strings.ToLower(q)
	idx := strings.Index(lq, "-m")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(q[idx+2:])
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "="))
	rest = strings.Trim(rest, "\"'")
	return rest
}

func nonEmpty(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func write(out output) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(out)
}
