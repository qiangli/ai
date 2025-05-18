package swarm

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os/exec"
	"reflect"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal/log"
	utool "github.com/qiangli/ai/internal/tool"
	"github.com/qiangli/ai/swarm/api"
)

// Default returns the given value if it's non-nil and non-zero value;
// otherwise, it returns the default value provided.
func Default(def, value any) any {
	v := reflect.ValueOf(value)
	if !v.IsValid() || reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface()) {
		return def
	}
	return value
}

// Spread concatenates the elements to create a single string.
func Spread(val any) string {
	if val == nil {
		return ""
	}
	var result = ""
	var items []string
	items, ok := val.([]string)
	if !ok {
		ar, ok := val.([]any)
		if ok {
			for _, v := range ar {
				if s, ok := v.(string); ok {
					items = append(items, s)
				} else {
					return fmt.Sprintf("%v", v)
				}
			}
		} else {
			return fmt.Sprintf("%v", val)
		}
	}

	for _, v := range items {
		if result != "" {
			result += " "
		}
		item := fmt.Sprintf("%v", v)
		// Escape double quotes and quote item if it contains spaces
		if strings.Contains(item, " ") {
			item = "\"" + strings.ReplaceAll(item, "\"", "\\\"") + "\""
		} else {
			item = strings.ReplaceAll(item, "\"", "\\\"")
		}

		result += item
	}
	return result
}

func callTplTool(ctx context.Context, vars *api.Vars, f *api.ToolFunc, args map[string]any) (string, error) {
	funcMap := map[string]any{
		"join":    strings.Join,
		"split":   strings.Split,
		"trim":    strings.TrimSpace,
		"default": Default,
		"spread":  Spread,
	}

	runCmd := func(cmd string, args ...string) (string, error) {
		result, err := execCommand(cmd, args, vars.Config.Debug)

		if err != nil {
			return result, err
		}
		if result == "" {
			return fmt.Sprintf("%s executed successfully", cmd), nil
		}
		return result, nil
	}

	// Add system commands to the function map
	for _, v := range toolSystemCommands {
		if _, err := exec.LookPath(v); err != nil {
			log.Errorf("%s not found in PATH\n", v)
			continue
		}
		funcMap[v] = func(args ...string) (string, error) {
			return runCmd(v, args...)
		}
	}
	funcMap["exec"] = runCmd

	var body string
	var err error
	if f.Body != "" {
		body, err = applyTemplate(f.Body, args, funcMap)
		if err != nil {
			return "", err
		}
	}

	switch f.Type {
	case ToolTypeTemplate:
		return body, nil
	case ToolTypeSql:
		cred, err := dbCred(vars, args)
		if err != nil {
			return "", err
		}
		return sqlQuery(ctx, cred, body)
	case ToolTypeShell:
		cmdline := strings.TrimSpace(body)
		return execCommand(cmdline, nil, vars.Config.Debug)
	}

	return "", fmt.Errorf("unknown function type %s for tool %s", f.Type, f.Name)
}

type SystemKit struct {
}

func callSystemTool(ctx context.Context, vars *api.Vars, f *api.ToolFunc, args map[string]any) (*api.Result, error) {
	tool := &SystemKit{}
	callArgs := []any{ctx, vars, f.Name, args}
	v, err := CallKit(tool, f.Kit, f.Name, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to call system tool %s %s: %w", f.Kit, f.Name, err)
	}

	// TODO change Value type to any?
	var result api.Result
	if s, ok := v.(string); ok {
		result.Value = s
	} else if c, ok := v.(*FileContent); ok {
		result.Value = string(c.Content)
		result.MimeType = c.MimeType
		result.Message = c.Message
	} else {
		result.Value = fmt.Sprintf("%v", v)
	}
	return &result, nil
}

func CallKit(tool any, kit string, method string, args ...any) (any, error) {
	instance := reflect.ValueOf(tool)
	name := toPascalCase(method)
	m := instance.MethodByName(name)
	if !m.IsValid() {
		return nil, fmt.Errorf("method %s not found on %s", method, kit)
	}

	if m.Type().NumIn() != len(args) {
		return nil, fmt.Errorf("wrong number of arguments for %s.%s", kit, method)
	}

	in := make([]reflect.Value, len(args))
	for i, arg := range args {
		in[i] = reflect.ValueOf(arg)
	}
	results := m.Call(in)

	if len(results) < 2 {
		return nil, fmt.Errorf("unexpected number of return values for %s.%s", kit, method)
	}

	v := results[0].Interface()
	var err error
	if !results[1].IsNil() {
		err = results[1].Interface().(error)
	}

	return v, err
}

func (r *SystemKit) getStr(key string, args map[string]any) (string, error) {
	return GetStrProp(key, args)
}

func (r *SystemKit) getArray(key string, args map[string]any) ([]string, error) {
	return GetArrayProp(key, args)
}

func (r *SystemKit) ListCommands(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	list := _os.ListCommands()
	return strings.Join(list, "\n"), nil
}

func (r *SystemKit) Which(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	commands, err := r.getArray("commands", args)
	if err != nil {
		return "", err
	}
	return _os.Which(commands)
}

func (r *SystemKit) Man(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	command, err := r.getStr("command", args)
	if err != nil {
		return "", err
	}
	return _os.Man(command)
}

func (r *SystemKit) Exec(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	command, err := r.getStr("command", args)
	if err != nil {
		return "", err
	}
	argsList, err := r.getArray("args", args)
	if err != nil {
		return "", err
	}
	return runRestricted(ctx, vars, command, argsList)
}

func (r *SystemKit) Cd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	dir, err := r.getStr("dir", args)
	if err != nil {
		return "", err
	}
	return "", _os.Chdir(dir)
}

func (r *SystemKit) Pwd(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return _os.Getwd()
}

func (r *SystemKit) Env(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return _os.Env(), nil
}

func (r *SystemKit) Uname(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	os, arch := _os.Uname()
	return fmt.Sprintf("OS: %s\nArch: %s", os, arch), nil
}

func (r *SystemKit) HomeDir(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.Home, nil
}
func (r *SystemKit) TempDir(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.Temp, nil
}
func (r *SystemKit) WorkspaceDir(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.Workspace, nil
}
func (r *SystemKit) RepoDir(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return vars.Repo, nil
}

func (r *SystemKit) ReadStdin(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return readStdin()
}

func (r *SystemKit) PasteFromClipboard(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return readClipboard()
}

func (r *SystemKit) PasteFromClipboardWait(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return readClipboardWait()
}

func (r *SystemKit) WriteStdout(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	content, err := r.getStr("content", args)
	if err != nil {
		return "", err
	}
	return writeStdout(content)
}

func (r *SystemKit) CopyToClipboard(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	content, err := r.getStr("content", args)
	if err != nil {
		return "", err
	}
	return writeClipboard(content)
}

func (r *SystemKit) CopyToClipboardAppend(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	content, err := r.getStr("content", args)
	if err != nil {
		return "", err
	}
	return writeClipboardAppend(content)
}

func (r *SystemKit) GetUserTextInput(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	prompt, err := r.getStr("prompt", args)
	if err != nil {
		return "", err
	}
	return getUserTextInput(prompt)
}

func (r *SystemKit) GetUserChoiceInput(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	prompt, err := r.getStr("prompt", args)
	if err != nil {
		return "", err
	}
	choices, err := r.getArray("choices", args)
	if err != nil {
		return "", err
	}
	defaultChoice, err := r.getStr("default", args)
	if err != nil {
		return "", err
	}
	return getUserChoiceInput(prompt, choices, defaultChoice)
}

type FuncKit struct {
}

func (r *FuncKit) WhatTimeIsIt(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return utool.WhatTimeIsIt()
}

func (r *FuncKit) WhoAmI(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return utool.WhoAmI()
}

func (r *FuncKit) FetchLocation(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	return utool.FetchLocation()
}

func (r *FuncKit) ListAgents(ctx context.Context, vars *api.Vars, _ string, _ map[string]any) (string, error) {
	var list []string
	dict := vars.ListAgents()
	for k, v := range dict {
		list = append(list, fmt.Sprintf("%s: %s", k, v.Description))
	}

	sort.Strings(list)
	return fmt.Sprintf("Available agents:\n%s\n", strings.Join(list, "\n")), nil
}

func (r *FuncKit) AgentInfo(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (string, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return "", err
	}
	dict := vars.ListAgents()
	if v, ok := dict[agent]; ok {
		return fmt.Sprintf("Agent: %s\nDescription: %s\n", v.Name, v.Description), nil
	}
	return "", fmt.Errorf("unknown agent: %s", agent)
}

// TODO
func callAgentTransfer(ctx context.Context, vars *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	dict := vars.ListAgents()
	if _, ok := dict[agent]; ok {
		return &api.Result{
			NextAgent: agent,
			State:     api.StateTransfer,
		}, nil
	}
	return nil, fmt.Errorf("unknown agent: %s", agent)
}

func (r *FuncKit) AskQuestion(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	question, err := GetStrProp("question", args)
	if err != nil {
		return "", err
	}
	return getUserTextInput(question)
}

func (r *FuncKit) TaskComplete(ctx context.Context, vars *api.Vars, name string, args map[string]any) (string, error) {
	log.Infof("✌️ task completed %s", name)
	return "Task completed", nil
}

func callFuncTool(ctx context.Context, vars *api.Vars, f *api.ToolFunc, args map[string]any) (string, error) {
	tool := &FuncKit{}
	callArgs := []any{ctx, vars, f.Name, args}
	v, err := CallKit(tool, f.Kit, f.Name, callArgs...)
	if err != nil {
		return "", fmt.Errorf("failed to call function tool %s %s: %w", f.Kit, f.Name, err)
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", v), nil
}
