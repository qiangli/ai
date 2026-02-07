package gitkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/qiangli/ai/swarm/api"
)

type Args struct {
	ID             string   `json:"id"`
	User           string   `json:"user"`
	Payload        string   `json:"payload"`
	Action         string   `json:"action"`
	Tool           string   `json:"tool"`
	Args           string   `json:"args"`
	Message        string   `json:"message"`
	Rev            string   `json:"rev"`
	Dir            string   `json:"dir"`
	Path           string   `json:"path"`
	ContextLines   int      `json:"context_lines"`
	Target         string   `json:"target"`
	MaxCount       int      `json:"max_count"`
	StartTimestamp string   `json:"start_timestamp"`
	EndTimestamp   string   `json:"end_timestamp"`
	BranchName     string   `json:"branch_name"`
	BaseBranch     string   `json:"base_branch"`
	BranchType     string   `json:"branch_type"`
	Contains       string   `json:"contains"`
	NotContains    string   `json:"not_contains"`
	Files          []string `json:"files"`
}

type payloadObj struct {
	Action         string   `json:"action,omitempty"`
	Tool           string   `json:"tool,omitempty"`
	Dir            string   `json:"dir,omitempty"`
	Args           []string `json:"args,omitempty"`
	Message        string   `json:"message,omitempty"`
	Rev            string   `json:"rev,omitempty"`
	Path           string   `json:"path,omitempty"`
	ContextLines   int      `json:"context_lines,omitempty"`
	Target         string   `json:"target,omitempty"`
	MaxCount       int      `json:"max_count,omitempty"`
	StartTimestamp string   `json:"start_timestamp,omitempty"`
	EndTimestamp   string   `json:"end_timestamp,omitempty"`
	BranchName     string   `json:"branch_name,omitempty"`
	BaseBranch     string   `json:"base_branch,omitempty"`
	BranchType     string   `json:"branch_type,omitempty"`
	Contains       string   `json:"contains,omitempty"`
	NotContains    string   `json:"not_contains,omitempty"`
	Files          []string `json:"files,omitempty"`
}

// Run is kept for backward compatibility and delegates to specific tool handlers.
func Run(args *Args) (any, error) {
	// Keep Run for backward compatibility: delegate to specific tool functions based on Args.Tool.
	tool := strings.ToLower(strings.TrimSpace(args.Tool))
	switch tool {
	case "git_status":
		return RunGitStatus(args)
	case "git_diff_unstaged":
		return RunGitDiffUnstaged(args)
	case "git_diff_staged":
		return RunGitDiffStaged(args)
	case "git_diff":
		return RunGitDiff(args)
	case "git_commit":
		return RunGitCommit(args)
	case "git_add":
		return RunGitAdd(args)
	case "git_reset":
		return RunGitReset(args)
	case "git_log":
		return RunGitLog(args)
	case "git_create_branch":
		return RunGitCreateBranch(args)
	case "git_checkout":
		return RunGitCheckout(args)
	case "git_show":
		return RunGitShow(args)
	case "git_branches":
		return RunGitBranches(args)
	default:
		return nil, fmt.Errorf("unsupported tool %q", args.Tool)
	}
}

// encodeOutput marshals Output to JSON string with HTML escaping disabled.
func encodeOutput(out Output) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(out); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// The following functions expose distinct tool handlers for each git operation.
// Each accepts *Args (as used by tests/wrappers) and returns the same (any,error) as Run.

func RunGitStatus(args *Args) (any, error) {
	outStr, errStr, err := Status(args.Dir)
	if outStr == "" {
		outStr = "clean"
	}
	out := Output{Stdout: outStr, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitDiffUnstaged(args *Args) (any, error) {
	ctx := 3
	if args.ContextLines != 0 {
		ctx = args.ContextLines
	}
	o, errStr, err := DiffUnstaged(args.Dir, ctx)
	out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitDiffStaged(args *Args) (any, error) {
	ctx := 3
	if args.ContextLines != 0 {
		ctx = args.ContextLines
	}
	o, errStr, err := DiffStaged(args.Dir, ctx)
	out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitDiff(args *Args) (any, error) {
	target := strings.TrimSpace(args.Target)
	if target == "" {
		target = args.Rev
	}
	if target == "" {
		out := Output{ExitCode: 2, OK: false, Error: "diff requires target"}
		return encodeOutput(out)
	}
	ctx := 3
	if args.ContextLines != 0 {
		ctx = args.ContextLines
	}
	o, errStr, err := DiffTarget(args.Dir, target, ctx)
	out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitCommit(args *Args) (any, error) {
	msg := strings.TrimSpace(args.Message)
	// build args.Args into slice if provided
	var extra []string
	if args.Args != "" && args.Args != "<no value>" {
		extra = api.ToStringArray(args.Args)
	}
	if msg == "" && len(extra) > 0 {
		msg = strings.Join(extra, " ")
	}
	if msg == "" {
		out := Output{ExitCode: 2, OK: false, Error: "commit requires message"}
		return encodeOutput(out)
	}
	stdout, stderr, code, err := Commit(args.Dir, msg, extra)
	out := Output{Stdout: stdout, Stderr: stderr, ExitCode: code, OK: err == nil}
	if err != nil {
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitAdd(args *Args) (any, error) {
	files := args.Files
	if len(files) == 0 {
		out := Output{ExitCode: 2, OK: false, Error: "add requires files"}
		return encodeOutput(out)
	}
	outStr, errStr, err := Add(args.Dir, files)
	out := Output{Stdout: outStr, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitReset(args *Args) (any, error) {
	o, errStr, err := Reset(args.Dir)
	out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitLog(args *Args) (any, error) {
	max := args.MaxCount
	startT := ParseTime(args.StartTimestamp)
	endT := ParseTime(args.EndTimestamp)
	logs, errStr, err := Log(args.Dir, max, startT, endT)
	out := Output{Stdout: "", Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
		out.Stdout = "[]"
	} else {
		bs, _ := json.Marshal(logs)
		out.Stdout = string(bs)
	}
	return encodeOutput(out)
}

func RunGitCreateBranch(args *Args) (any, error) {
	name := strings.TrimSpace(args.BranchName)
	if name == "" {
		out := Output{ExitCode: 2, OK: false, Error: "create-branch requires branch_name"}
		return encodeOutput(out)
	}
	o, errStr, err := CreateBranch(args.Dir, name, args.BaseBranch)
	out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitCheckout(args *Args) (any, error) {
	name := strings.TrimSpace(args.BranchName)
	if name == "" {
		out := Output{ExitCode: 2, OK: false, Error: "checkout requires branch_name"}
		return encodeOutput(out)
	}
	o, errStr, err := Checkout(args.Dir, name)
	out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitShow(args *Args) (any, error) {
	rev := strings.TrimSpace(args.Rev)
	if rev == "" && args.Args != "" {
		arr := api.ToStringArray(args.Args)
		if len(arr) > 0 {
			rev = arr[0]
		}
	}
	if rev == "" {
		out := Output{ExitCode: 2, OK: false, Error: "show requires rev"}
		return encodeOutput(out)
	}
	o, errStr, err := Show(args.Dir, rev)
	out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

func RunGitBranches(args *Args) (any, error) {
	typ := strings.ToLower(strings.TrimSpace(args.BranchType))
	o, errStr, err := Branches(args.Dir, typ)
	out := Output{Stdout: o, Stderr: errStr, ExitCode: 0, OK: err == nil}
	if err != nil {
		out.ExitCode = 1
		out.Error = err.Error()
	}
	return encodeOutput(out)
}

// Output represents the standardized tool output
type Output struct {
	ID       string `json:"id,omitempty"`
	User     string `json:"user,omitempty"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	OK       bool   `json:"ok"`
	Error    string `json:"error,omitempty"`
}
