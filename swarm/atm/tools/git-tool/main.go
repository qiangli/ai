package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/qiangli/ai/swarm/atm/gitkit"
)

// Envelope is the preferred request format.
// Backward compatibility: callers may also send a bare payload object.
//
// Example:
//  {"id":"1","user":"me","payload":{"action":"status","dir":"/repo"}}
//
// Bare payload example:
//  {"action":"status","dir":"/repo"}
//  {"command":["git","status"],"dir":"/repo"}
//
// Notes:
// - This tool is not a user-facing CLI. It is intended to be invoked by agent tools.
// - It does not execute an arbitrary shell command.

type Envelope struct {
	ID      string          `json:"id,omitempty"`
	User    string          `json:"user,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type Payload struct {
	// Raw command mode.
	Command []string `json:"command,omitempty"` // must start with "git"

	// Structured action mode.
	Action  string   `json:"action,omitempty"`
	Dir     string   `json:"dir,omitempty"`
	Args    []string `json:"args,omitempty"`
	Message string   `json:"message,omitempty"` // commit convenience

	// Convenience fields for specific actions.
	Rev  string `json:"rev,omitempty"`  // rev-parse/show-file
	Path string `json:"path,omitempty"` // show-file
}

type Output struct {
	ID       string `json:"id,omitempty"`
	User     string `json:"user,omitempty"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
}

func main() {
	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		writeAndExit(Output{ExitCode: 2, OK: false, Error: fmt.Sprintf("read stdin: %v", err)}, 2)
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		writeAndExit(Output{ExitCode: 2, OK: false, Error: "empty stdin; expected JSON"}, 2)
	}

	env, payload, err := decode(raw)
	if err != nil {
		writeAndExit(Output{ID: env.ID, User: env.User, ExitCode: 2, OK: false, Error: err.Error()}, 2)
	}

	stdout, stderr, exitCode, runErr := run(payload)
	out := Output{ID: env.ID, User: env.User, Stdout: stdout, Stderr: stderr, ExitCode: exitCode, OK: runErr == nil}
	if runErr != nil {
		out.Error = runErr.Error()
		writeAndExit(out, exitCode)
	}
	writeAndExit(out, 0)
}

func decode(raw []byte) (Envelope, Payload, error) {
	var env Envelope
	// First try envelope.
	if err := json.Unmarshal(raw, &env); err == nil && len(env.Payload) > 0 {
		var p Payload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			return env, Payload{}, fmt.Errorf("decode envelope payload: %w", err)
		}
		return env, p, nil
	}
	// Fallback: bare payload.
	env = Envelope{}
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return env, Payload{}, fmt.Errorf("decode payload: %w", err)
	}
	return env, p, nil
}

func run(p Payload) (stdout string, stderr string, exitCode int, err error) {
	if len(p.Command) > 0 {
		if err := validateCommand(p.Command); err != nil {
			return "", "", 2, err
		}
		out, errOut, code, runErr := gitkit.RunGitExitCode(p.Dir, p.Command[1:]...)
		return out, errOut, code, runErr
	}

	action := strings.ToLower(strings.TrimSpace(p.Action))
	switch action {
	case "status":
		outStr, errStr, runErr := gitkit.Status(p.Dir)
		if runErr != nil {
			return outStr, errStr, 1, runErr
		}
		return outStr, errStr, 0, nil
	case "clone":
		if len(p.Args) != 2 {
			return "", "", 2, errors.New("clone requires args: [repoURL, destDir]")
		}
		err := gitkit.Clone(p.Args[0], p.Args[1])
		if err != nil {
			return "", "", 1, err
		}
		return "", "", 0, nil
	case "commit":
		msg := strings.TrimSpace(p.Message)
		if msg == "" && len(p.Args) > 0 {
			msg = strings.Join(p.Args, " ")
		}
		if strings.TrimSpace(msg) == "" {
			return "", "", 2, errors.New("commit requires non-empty message")
		}
		out, errOut, code, runErr := gitkit.RunGitExitCode(p.Dir, "commit", "-m", msg)
		return out, errOut, code, runErr
	case "pull":
		out, errOut, code, runErr := gitkit.RunGitExitCode(p.Dir, append([]string{"pull"}, p.Args...)...)
		return out, errOut, code, runErr
	case "push":
		out, errOut, code, runErr := gitkit.RunGitExitCode(p.Dir, append([]string{"push"}, p.Args...)...)
		return out, errOut, code, runErr
	case "branch", "current-branch":
		branch, errOut, runErr := gitkit.CurrentBranch(p.Dir)
		if runErr != nil {
			return branch, errOut, 1, runErr
		}
		return branch, errOut, 0, nil
	case "remote-url":
		url, errOut, runErr := gitkit.RemoteURL(p.Dir)
		if runErr != nil {
			return url, errOut, 1, runErr
		}
		return url, errOut, 0, nil
	case "rev-parse":
		rev := strings.TrimSpace(p.Rev)
		if rev == "" && len(p.Args) == 1 {
			rev = p.Args[0]
		}
		if rev == "" {
			return "", "", 2, errors.New("rev-parse requires rev (field 'rev' or args[0])")
		}
		hash, errOut, runErr := gitkit.RevParse(p.Dir, rev)
		if runErr != nil {
			return hash, errOut, 1, runErr
		}
		return hash, errOut, 0, nil
	case "list-branches":
		out, errOut, runErr := gitkit.ListBranches(p.Dir)
		if runErr != nil {
			return out, errOut, 1, runErr
		}
		return out, errOut, 0, nil
	case "list-remotes":
		out, errOut, runErr := gitkit.ListRemotes(p.Dir)
		if runErr != nil {
			return out, errOut, 1, runErr
		}
		return out, errOut, 0, nil
	case "latest-commit":
		out, errOut, runErr := gitkit.LatestCommit(p.Dir)
		if runErr != nil {
			return out, errOut, 1, runErr
		}
		return out, errOut, 0, nil
	case "show-file":
		rev := strings.TrimSpace(p.Rev)
		path := strings.TrimSpace(p.Path)
		if rev == "" && len(p.Args) >= 1 {
			rev = p.Args[0]
		}
		if path == "" && len(p.Args) >= 2 {
			path = p.Args[1]
		}
		if rev == "" || path == "" {
			return "", "", 2, errors.New("show-file requires rev and path (fields 'rev' and 'path' or args[0]=rev,args[1]=path)")
		}
		out, errOut, runErr := gitkit.ShowFileAtRev(p.Dir, rev, path)
		if runErr != nil {
			return out, errOut, 1, runErr
		}
		return out, errOut, 0, nil
	case "raw":
		if len(p.Args) == 0 {
			return "", "", 2, errors.New("raw requires args: full git argv including 'git'")
		}
		if err := validateCommand(p.Args); err != nil {
			return "", "", 2, err
		}
		out, errOut, code, runErr := gitkit.RunGitExitCode(p.Dir, p.Args[1:]...)
		return out, errOut, code, runErr
	default:
		return "", "", 2, fmt.Errorf("unsupported action %q", action)
	}
}

func validateCommand(cmd []string) error {
	if len(cmd) == 0 {
		return errors.New("command must be a non-empty array")
	}
	if cmd[0] != "git" {
		return errors.New("command[0] must be 'git'")
	}
	return nil
}

func writeAndExit(out Output, code int) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(out)
	os.Exit(code)
}
