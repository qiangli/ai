// package sh
package gitkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api"
)

// Top-level fields (strings) injected by the runtime.
type Args struct {
	ID      string `json:"id"`
	User    string `json:"user"`
	Payload string `json:"payload"` // JSON object
	Action  string `json:"action"`
	Dir     string `json:"dir"`
	Args    string `json:"args"` // JSON array
	Message string `json:"message"`
	Rev     string `json:"rev"`
	Path    string `json:"path"`
	Command string `json:"command"` // JSON array
}

// var id = "<no value>"
// var user = "<no value>"
// var payload = "<no value>" // JSON object

// var action = "<no value>"
// var dir = "<no value>"
// var args = "<no value>" // JSON array
// var message = "<no value>"
// var rev = "<no value>"
// var path = "<no value>"
// var command = "<no value>" // JSON array

type payloadObj struct {
	Action  string   `json:"action,omitempty"`
	Dir     string   `json:"dir,omitempty"`
	Args    []string `json:"args,omitempty"`
	Message string   `json:"message,omitempty"`
	Rev     string   `json:"rev,omitempty"`
	Path    string   `json:"path,omitempty"`
	Command []string `json:"command,omitempty"`
}

type envelope struct {
	ID      string     `json:"id,omitempty"`
	User    string     `json:"user,omitempty"`
	Payload payloadObj `json:"payload"`
}

func Run(args *Args) (any, error) {
	var env envelope
	if args.ID != "" && args.ID != "<no value>" {
		env.ID = args.ID
	}
	if args.User != "" && args.User != "<no value>" {
		env.User = args.User
	}

	// Prefer explicit payload.
	if args.Payload != "" && args.Payload != "<no value>" {
		if err := json.Unmarshal([]byte(args.Payload), &env.Payload); err != nil {
			return nil, fmt.Errorf("invalid payload JSON: %v\n", err)
		}
	} else {
		// Build from top-level convenience fields.
		if args.Action != "" && args.Action != "<no value>" {
			env.Payload.Action = args.Action
		}
		if args.Dir != "" && args.Dir != "<no value>" {
			env.Payload.Dir = args.Dir
		}
		if args.Message != "" && args.Message != "<no value>" {
			env.Payload.Message = args.Message
		}
		if args.Rev != "" && args.Rev != "<no value>" {
			env.Payload.Rev = args.Rev
		}
		if args.Path != "" && args.Path != "<no value>" {
			env.Payload.Path = args.Path
		}

		if args.Args != "" && args.Args != "<no value>" {
			var list []string
			// if err := json.Unmarshal([]byte(args.Args), &list); err != nil {
			// 	return nil, fmt.Errorf("invalid args JSON: %v\n", err)
			// }
			list = api.ToStringArray(args.Args)
			env.Payload.Args = list
		}
		if args.Command != "" && args.Command != "<no value>" {
			var list []string
			// if err := json.Unmarshal([]byte(args.Command), &list); err != nil {
			// 	return nil, fmt.Errorf("invalid command JSON: %v\n", err)
			// }
			list = api.ToStringArray(args.Command)
			env.Payload.Command = list
		}
	}

	// Basic validation before invoking.
	if len(env.Payload.Command) > 0 {
		if env.Payload.Command[0] != "git" {
			return nil, fmt.Errorf("command must start with 'git'\n")
		}
	} else {
		a := strings.TrimSpace(env.Payload.Action)
		if a == "" {
			return nil, fmt.Errorf("either payload.command or payload.action is required\n")
		}
	}

	// Now dispatch using gitkit directly to avoid spawning external processes.
	out := run(env.Payload)

	// if !out.OK {
	// 	// Use a non-zero exit code when tool reported error.
	// 	return nil, fmt.Errorf("Error: %v, Exit code: %v", out.Error, out.ExitCode)
	// }

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	err := enc.Encode(out)
	if err != nil {
		return nil, err
	}

	return buf.String(), nil
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

func run(p payloadObj) Output {
	// Raw command mode.
	if len(p.Command) > 0 {
		stdout, stderr, code, err := RunGitExitCode(p.Dir, p.Command[1:]...)
		out := Output{Stdout: stdout, Stderr: stderr, ExitCode: code, OK: err == nil}
		if err != nil {
			out.Error = err.Error()
		}
		return out
	}

	action := strings.ToLower(strings.TrimSpace(p.Action))
	switch action {
	case "status":
		outStr, errStr, err := Status(p.Dir)
		out := Output{Stdout: outStr, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "clone":
		if len(p.Args) != 2 {
			return Output{ExitCode: 2, OK: false, Error: "clone requires args: [repoURL, destDir]"}
		}
		err := Clone(p.Args[0], p.Args[1])
		if err != nil {
			return Output{ExitCode: 1, OK: false, Error: err.Error()}
		}
		return Output{ExitCode: 0, OK: true}
	case "commit":
		msg := strings.TrimSpace(p.Message)
		if msg == "" && len(p.Args) > 0 {
			msg = strings.Join(p.Args, " ")
		}
		if strings.TrimSpace(msg) == "" {
			return Output{ExitCode: 2, OK: false, Error: "commit requires non-empty message"}
		}
		stdout, stderr, code, err := RunGitExitCode(p.Dir, "commit", "-m", msg)
		out := Output{Stdout: stdout, Stderr: stderr, ExitCode: code, OK: err == nil}
		if err != nil {
			out.Error = err.Error()
		}
		return out
	case "pull":
		stdout, stderr, code, err := RunGitExitCode(p.Dir, append([]string{"pull"}, p.Args...)...)
		out := Output{Stdout: stdout, Stderr: stderr, ExitCode: code, OK: err == nil}
		if err != nil {
			out.Error = err.Error()
		}
		return out
	case "push":
		stdout, stderr, code, err := RunGitExitCode(p.Dir, append([]string{"push"}, p.Args...)...)
		out := Output{Stdout: stdout, Stderr: stderr, ExitCode: code, OK: err == nil}
		if err != nil {
			out.Error = err.Error()
		}
		return out
	case "branch", "current-branch":
		b, errStr, err := CurrentBranch(p.Dir)
		out := Output{Stdout: b, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "remote-url":
		u, errStr, err := RemoteURL(p.Dir)
		out := Output{Stdout: u, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "rev-parse":
		rev := strings.TrimSpace(p.Rev)
		if rev == "" && len(p.Args) == 1 {
			rev = p.Args[0]
		}
		if rev == "" {
			return Output{ExitCode: 2, OK: false, Error: "rev-parse requires rev (field 'rev' or args[0])"}
		}
		h, errStr, err := RevParse(p.Dir, rev)
		out := Output{Stdout: h, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "list-branches":
		o, errStr, err := ListBranches(p.Dir)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "list-remotes":
		o, errStr, err := ListRemotes(p.Dir)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "latest-commit":
		o, errStr, err := LatestCommit(p.Dir)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
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
			return Output{ExitCode: 2, OK: false, Error: "show-file requires rev and path (fields 'rev' and 'path' or args[0]=rev,args[1]=path)"}
		}
		o, errStr, err := ShowFileAtRev(p.Dir, rev, path)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "raw":
		if len(p.Args) == 0 {
			return Output{ExitCode: 2, OK: false, Error: "raw requires args: full git argv including 'git'"}
		}
		if p.Args[0] != "git" {
			return Output{ExitCode: 2, OK: false, Error: "raw command must start with 'git'"}
		}
		stdout, stderr, code, err := RunGitExitCode(p.Dir, p.Args[1:]...)
		out := Output{Stdout: stdout, Stderr: stderr, ExitCode: code, OK: err == nil}
		if err != nil {
			out.Error = err.Error()
		}
		return out
	default:
		return Output{ExitCode: 2, OK: false, Error: fmt.Sprintf("unsupported action %q", action)}
	}
}
