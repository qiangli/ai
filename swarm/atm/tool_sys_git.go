package atm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/gitkit"
)

type GitKit struct {
}

func (r *GitKit) Call(ctx context.Context, vars *api.Vars, agent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	callArgs := []any{ctx, vars, agent, tf, args}
	v, err := CallKit(r, tf.Kit, tf.Name, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s:%s error: %w", tf.Kit, tf.Name, err)
	}
	return v, err
}

func (r *GitKit) Status(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	resp, _, err := gitkit.Status(dir)
	return resp, err
}

func (r *GitKit) DiffUnstaged(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	contextLines, _ := api.GetIntProp("context_lines", args)
	gitArgs := &gitkit.Args{
		Dir:          dir,
		ContextLines: contextLines,
	}
	return gitkit.RunGitDiffUnstaged(gitArgs)
}

func (r *GitKit) DiffStaged(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	contextLines, _ := api.GetIntProp("context_lines", args)
	gitArgs := &gitkit.Args{
		Dir:          dir,
		ContextLines: contextLines,
	}
	return gitkit.RunGitDiffStaged(gitArgs)
}

func (r *GitKit) Diff(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	target, err := api.GetStrProp("target", args)
	if err != nil {
		return nil, fmt.Errorf("target is required: %w", err)
	}
	contextLines, _ := api.GetIntProp("context_lines", args)
	gitArgs := &gitkit.Args{
		Dir:          dir,
		Target:       target,
		ContextLines: contextLines,
	}
	return gitkit.RunGitDiff(gitArgs)
}

func (r *GitKit) Commit(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	message, err := api.GetStrProp("message", args)
	if err != nil {
		return nil, fmt.Errorf("message is required: %w", err)
	}
	// handle optional args []string
	argList := []string{}
	if argIface, ok := args["args"]; ok {
		if argListI, ok := argIface.([]interface{}); ok {
			argList = make([]string, len(argListI))
			for i, a := range argListI {
				if as, ok := a.(string); ok {
					argList[i] = as
				}
			}
		}
	}
	argJSON, _ := json.Marshal(argList)
	gitArgs := &gitkit.Args{
		Dir:     dir,
		Message: message,
		Args:    string(argJSON),
	}
	return gitkit.RunGitCommit(gitArgs)
}

func (r *GitKit) Add(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	// filesIface, ok := args["files"]
	// if !ok {
	// 	return nil, fmt.Errorf("files is required")
	// }
	// filesListI, ok := filesIface.([]any)
	// if !ok || len(filesListI) == 0 {
	// 	return nil, fmt.Errorf("files is required and must be non-empty array")
	// }
	// files := make([]string, len(filesListI))
	// for i, f := range filesListI {
	// 	if fs, ok := f.(string); ok {
	// 		files[i] = fs
	// 	} else {
	// 		return nil, fmt.Errorf("files[%d] must be string", i)
	// 	}
	// }
	// filesJSON, _ := json.Marshal(files)
	files, err := api.GetArrayProp("files", args)
	if err != nil {
		return nil, err
	}
	gitArgs := &gitkit.Args{
		Dir:   dir,
		Files: files,
	}
	return gitkit.RunGitAdd(gitArgs)
}

func (r *GitKit) Reset(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	gitArgs := &gitkit.Args{
		Dir: dir,
	}
	return gitkit.RunGitReset(gitArgs)
}

func (r *GitKit) Log(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	maxCount, _ := api.GetIntProp("max_count", args)
	startTS, _ := api.GetStrProp("start_timestamp", args)
	endTS, _ := api.GetStrProp("end_timestamp", args)
	gitArgs := &gitkit.Args{
		Dir:            dir,
		MaxCount:       maxCount,
		StartTimestamp: startTS,
		EndTimestamp:   endTS,
	}
	return gitkit.RunGitLog(gitArgs)
}

func (r *GitKit) CreateBranch(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	branchName, err := api.GetStrProp("branch_name", args)
	if err != nil {
		return nil, fmt.Errorf("branch_name is required: %w", err)
	}
	baseBranch, _ := api.GetStrProp("base_branch", args)
	gitArgs := &gitkit.Args{
		Dir:        dir,
		BranchName: branchName,
		BaseBranch: baseBranch,
	}
	return gitkit.RunGitCreateBranch(gitArgs)
}

func (r *GitKit) Checkout(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	branchName, err := api.GetStrProp("branch_name", args)
	if err != nil {
		return nil, fmt.Errorf("branch_name is required: %w", err)
	}
	gitArgs := &gitkit.Args{
		Dir:        dir,
		BranchName: branchName,
	}
	return gitkit.RunGitCheckout(gitArgs)
}

func (r *GitKit) Show(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	rev, err := api.GetStrProp("rev", args)
	if err != nil {
		return nil, fmt.Errorf("rev is required: %w", err)
	}
	gitArgs := &gitkit.Args{
		Dir: dir,
		Rev: rev,
	}
	return gitkit.RunGitShow(gitArgs)
}

func (r *GitKit) Branches(ctx context.Context, vars *api.Vars, parent *api.Agent, tf *api.ToolFunc, args map[string]any) (any, error) {
	dir, err := api.GetStrProp("dir", args)
	if err != nil {
		return nil, err
	}
	branchType, _ := api.GetStrProp("branch_type", args)
	gitArgs := &gitkit.Args{
		Dir:        dir,
		BranchType: branchType,
	}
	return gitkit.RunGitBranches(gitArgs)
}
