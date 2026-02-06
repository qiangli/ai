package gitkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api"
)

type Args struct {
	ID            string `json:"id"`
	User          string `json:"user"`
	Payload       string `json:"payload"`
	Action        string `json:"action"`
	Args          string `json:"args"`
	Message       string `json:"message"`
	Rev           string `json:"rev"`
	Path          string `json:"path"`
	ContextLines  int    `json:"context_lines"`
	Target        string `json:"target"`
	MaxCount      int    `json:"max_count"`
	StartTimestamp string `json:"start_timestamp"`
	EndTimestamp  string `json:"end_timestamp"`
	BranchName    string `json:"branch_name"`
	BaseBranch    string `json:"base_branch"`
	BranchType    string `json:"branch_type"`
	Contains      string `json:"contains"`
	NotContains   string `json:"not_contains"`
	Files         string `json:"files"`
}

type payloadObj struct {
	Action       string   `json:"action,omitempty"`
	Dir          string   `json:"dir,omitempty"`
	Args         []string `json:"args,omitempty"`
	Message      string   `json:"message,omitempty"`
	Rev          string   `json:"rev,omitempty"`
	Path         string   `json:"path,omitempty"`
	Command      []string `json:"command,omitempty"`
	ContextLines int      `json:"context_lines,omitempty"`
	Target       string   `json:"target,omitempty"`
	MaxCount     int      `json:"max_count,omitempty"`
	StartTimestamp string `json:"start_timestamp,omitempty"`
	EndTimestamp string  `json:"end_timestamp,omitempty"`
	BranchName   string   `json:"branch_name,omitempty"`
	BaseBranch   string   `json:"base_branch,omitempty"`
	BranchType   string   `json:"branch_type,omitempty"`
	Contains     string   `json:"contains,omitempty"`
	NotContains  string   `json:"not_contains,omitempty"`
	Files        []string `json:"files,omitempty"`
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

	if args.Payload != "" && args.Payload != "<no value>" {
		if err := json.Unmarshal([]byte(args.Payload), &env.Payload); err != nil {
			return nil, fmt.Errorf("invalid payload JSON: %w", err)
		}
	} else {
		if args.Action != "" && args.Action != "<no value>" {
			env.Payload.Action = args.Action
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
		if args.Target != "" && args.Target != "<no value>" {
			env.Payload.Target = args.Target
		}
		if args.BranchName != "" && args.BranchName != "<no value>" {
			env.Payload.BranchName = args.BranchName
		}
		if args.BaseBranch != "" && args.BaseBranch != "<no value>" {
			env.Payload.BaseBranch = args.BaseBranch
		}
		if args.BranchType != "" && args.BranchType != "<no value>" {
			env.Payload.BranchType = args.BranchType
		}
		if args.Contains != "" && args.Contains != "<no value>" {
			env.Payload.Contains = args.Contains
		}
		if args.NotContains != "" && args.NotContains != "<no value>" {
			env.Payload.NotContains = args.NotContains
		}
		env.Payload.ContextLines = args.ContextLines
		env.Payload.MaxCount = args.MaxCount
		if args.Args != "" && args.Args != "<no value>" {
			env.Payload.Args = api.ToStringArray(args.Args)
		}
		if args.Files != "" && args.Files != "<no value>" {
			env.Payload.Files = api.ToStringArray(args.Files)
		}
		if args.StartTimestamp != "" && args.StartTimestamp != "<no value>" {
			env.Payload.StartTimestamp = args.StartTimestamp
		}
		if args.EndTimestamp != "" && args.EndTimestamp != "<no value>" {
			env.Payload.EndTimestamp = args.EndTimestamp
		}
	}

	if len(env.Payload.Command) > 0 {
		if env.Payload.Command[0] != "git" {
			return nil, fmt.Errorf("command must start with 'git'")
		}
	} else {
		a := strings.TrimSpace(env.Payload.Action)
		if a == "" {
			return nil, fmt.Errorf("either payload.command or payload.action is required")
		}
	}

	out := run(env.Payload)

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
	case "add":
		files := p.Args
		if len(p.Files) > 0 {
			files = p.Files
		}
		if len(files) == 0 {
			return Output{ExitCode: 2, OK: false, Error: "add requires files/args"}
		}
		outStr, errStr, err := Add(p.Dir, files)
		out := Output{Stdout: outStr, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "commit":
		msg := strings.TrimSpace(p.Message)
		if msg == "" && len(p.Args) > 0 {
			msg = strings.Join(p.Args, " ")
		}
		if msg == "" {
			return Output{ExitCode: 2, OK: false, Error: "commit requires message"}
		}
		stdout, stderr, code, err := Commit(p.Dir, msg, p.Args)
		out := Output{Stdout: stdout, Stderr: stderr, ExitCode: code, OK: err == nil}
		if err != nil {
			out.Error = err.Error()
		}
		return out
	case "pull":
		outStr, errStr, err := Pull(p.Dir, p.Args)
		out := Output{Stdout: outStr, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "push":
		outStr, errStr, err := Push(p.Dir, p.Args)
		out := Output{Stdout: outStr, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
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
			return Output{ExitCode: 2, OK: false, Error: "rev-parse requires rev"}
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
			return Output{ExitCode: 2, OK: false, Error: "show-file requires rev and path"}
		}
		o, errStr, err := ShowFileAtRev(p.Dir, rev, path)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "diff-unstaged":
		ctx := 3
		if p.ContextLines != 0 {
			ctx = p.ContextLines
		}
		o, errStr, err := DiffUnstaged(p.Dir, ctx)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "diff-staged":
		ctx := 3
		if p.ContextLines != 0 {
			ctx = p.ContextLines
		}
		o, errStr, err := DiffStaged(p.Dir, ctx)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "diff":
		target := strings.TrimSpace(p.Target)
		if target == "" {
			target = p.Rev
		}
		if target == "" {
			return Output{ExitCode: 2, OK: false, Error: "diff requires target or rev"}
		}
		ctx := 3
		if p.ContextLines != 0 {
			ctx = p.ContextLines
		}
		o, errStr, err := DiffTarget(p.Dir, target, ctx)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "reset":
		o, errStr, err := Reset(p.Dir)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "log":
		max := p.MaxCount
		startT := ParseTime(p.StartTimestamp)
		endT := ParseTime(p.EndTimestamp)
		logs, errStr, err := Log(p.Dir, max, startT, endT)
		out := Output{Stdout: "", Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
			out.Stdout = "[]"
		} else {
			bs, _ := json.Marshal(logs)
			out.Stdout = string(bs)
		}
		return out
	case "create-branch":
		name := strings.TrimSpace(p.BranchName)
		if name == "" {
			return Output{ExitCode: 2, OK: false, Error: "create-branch requires branch_name"}
		}
		o, errStr, err := CreateBranch(p.Dir, name, p.BaseBranch)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "checkout":
		name := strings.TrimSpace(p.BranchName)
		if name == "" {
			return Output{ExitCode: 2, OK: false, Error: "checkout requires branch_name"}
		}
		o, errStr, err := Checkout(p.Dir, name)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "show":
		rev := strings.TrimSpace(p.Rev)
		if rev == "" && len(p.Args) > 0 {
			rev = p.Args[0]
		}
		if rev == "" {
			return Output{ExitCode: 2, OK: false, Error: "show requires rev"}
		}
		o, errStr, err := Show(p.Dir, rev)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "branches":
		typ := strings.ToLower(strings.TrimSpace(p.BranchType))
		o, errStr, err := Branches(p.Dir, typ)
		out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
		if err != nil {
			out.ExitCode = 1
			out.Error = err.Error()
		}
		return out
	case "raw":
		if len(p.Args) == 0 {
			return Output{ExitCode: 2, OK: false, Error: "raw requires args"}
		}
		if p.Args[0] != "git" {
			return Output{ExitCode: 2, OK: false, Error: "raw must start with 'git'"}
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


