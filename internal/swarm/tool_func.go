package swarm

import (
	"context"
	"fmt"
	"os/exec"
	"reflect"
	"strings"

	"github.com/qiangli/ai/internal/log"
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

func callDevTool(ctx context.Context, vars *Vars, f *ToolFunc, args map[string]any) (string, error) {
	funcMap := map[string]any{
		"join":    strings.Join,
		"split":   strings.Split,
		"trim":    strings.TrimSpace,
		"default": Default,
	}

	runCmd := func(cmd string, args ...any) string {
		nArgs := []string{}
		for _, arg := range args {
			switch v := arg.(type) {
			case string:
				v = strings.TrimSpace(v)
				if v != "" {
					nArgs = append(nArgs, v)
				}
			case []string:
				for _, s := range v {
					s = strings.TrimSpace(s)
					if s != "" {
						nArgs = append(nArgs, s)
					}
				}
			case []any:
				for _, item := range v {
					switch i := item.(type) {
					case string:
						i = strings.TrimSpace(i)
						if i != "" {
							nArgs = append(nArgs, i)
						}
					default:
						log.Errorf("Unsupported item type in []interface{} for command %s: %T", cmd, i)
					}
				}
			default:
				log.Errorf("Unsupported argument type for command %s: %T", cmd, v)
			}
		}
		log.Debugf("Running command: %s %+v original: %+v\n", cmd, nArgs, args)
		result, err := runCommand(cmd, nArgs)
		if err != nil {
			return fmt.Sprintf("%s %s: %s", cmd, strings.Join(nArgs, " "), err.Error())
		}
		if result == "" {
			return fmt.Sprintf("%s executed successfully", cmd)
		}
		return result
	}

	// Add system commands to the function map
	for _, v := range toolSystemCommands {
		if _, err := exec.LookPath(v); err != nil {
			log.Errorf("%s not found in PATH", v)
			continue
		}
		funcMap[v] = func(args ...any) string {
			return runCmd(v, args...)
		}
	}
	funcMap["exec"] = runCmd

	var body string
	var err error
	if f.Body != "" {
		body, err = applyTemplate(f.Body, args, funcMap)
		if err != nil {
			return "", fmt.Errorf("failed to apply function body %s: %w", f.Name, err)
		}
	}

	switch f.Type {
	case "template":
		return body, nil
	case "shell":
		return runRestricted(ctx, vars, body, []string{})
	}

	return "", fmt.Errorf("unknown function type %s for tool %s", f.Type, f.Name)
}

type SystemKit struct {
}

func callSystemTool(ctx context.Context, vars *Vars, f *ToolFunc, args map[string]any) (string, error) {
	tool := &SystemKit{}
	callArgs := []any{ctx, vars, f.Name, args}
	v, err := CallKit(tool, f.Kit, f.Name, callArgs...)
	if err != nil {
		return "", fmt.Errorf("failed to call system tool %s %s: %w", f.Kit, f.Name, err)
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return fmt.Sprintf("%v", v), nil
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

// func callSystemfunc(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
func (r *SystemKit) getStr(key string, args map[string]any) (string, error) {
	return GetStrProp(key, args)
}

func (r *SystemKit) getArray(key string, args map[string]any) ([]string, error) {
	return GetArrayProp(key, args)
}

func (r *SystemKit) ListDirectory(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	list, err := _fs.ListDirectory(path)
	if err != nil {
		return "", err
	}
	return strings.Join(list, "\n"), nil
}

func (r *SystemKit) CreateDirectory(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	return "", _fs.CreateDirectory(path)
}

func (r *SystemKit) RenameFile(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	source, err := r.getStr("source", args)
	if err != nil {
		return "", err
	}
	dest, err := r.getStr("destination", args)
	if err != nil {
		return "", err
	}
	if err := _fs.RenameFile(source, dest); err != nil {
		return "", err
	}
	return "File renamed successfully", nil
}

func (r *SystemKit) GetFileInfo(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	info, err := _fs.GetFileInfo(path)
	if err != nil {
		return "", err
	}
	return info.String(), nil
}

func (r *SystemKit) ReadFile(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	content, err := _fs.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (r *SystemKit) WriteFile(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	path, err := r.getStr("path", args)
	if err != nil {
		return "", err
	}
	content, err := r.getStr("content", args)
	if err != nil {
		return "", err
	}
	if err := _fs.WriteFile(path, []byte(content)); err != nil {
		return "", err
	}
	return "File written successfully", nil
}

func (r *SystemKit) ListCommands(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	list := _os.ListCommands()
	return strings.Join(list, "\n"), nil
}

func (r *SystemKit) Which(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	commands, err := r.getArray("commands", args)
	if err != nil {
		return "", err
	}
	return _os.Which(commands)
}

func (r *SystemKit) Man(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	command, err := r.getStr("command", args)
	if err != nil {
		return "", err
	}
	return _os.Man(command)
}

func (r *SystemKit) Exec(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
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

func (r *SystemKit) Cd(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	dir, err := r.getStr("dir", args)
	if err != nil {
		return "", err
	}
	return "", _os.Chdir(dir)
}

func (r *SystemKit) Pwd(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	return workDir()
}

func (r *SystemKit) Env(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	return _os.Env(), nil
}

func (r *SystemKit) Uname(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	os, arch := _os.Uname()
	return fmt.Sprintf("OS: %s\nArch: %s", os, arch), nil
}

func (r *SystemKit) HomeDir(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	return homeDir()
}
func (r *SystemKit) TempDir(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	return tempDir()
}
func (r *SystemKit) WorkspaceDir(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	return resolveWorkspaceDir(vars.Workspace)
}
func (r *SystemKit) RepoDir(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	return resolveRepoDir()
}

func (r *SystemKit) ReadStdin(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	return readStdin()
}

func (r *SystemKit) PasteFromClipboard(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	return readClipboard()
}

func (r *SystemKit) PasteFromClipboardWait(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	return readClipboardWait()
}

func (r *SystemKit) WriteStdout(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	content, err := r.getStr("content", args)
	if err != nil {
		return "", err
	}
	return writeStdout(content)
}

func (r *SystemKit) CopyToClipboard(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	content, err := r.getStr("content", args)
	if err != nil {
		return "", err
	}
	return writeClipboard(content)
}

func (r *SystemKit) CopyToClipboardAppend(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	content, err := r.getStr("content", args)
	if err != nil {
		return "", err
	}
	return writeClipboardAppend(content)
}

func (r *SystemKit) GetUserTextInput(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
	prompt, err := r.getStr("prompt", args)
	if err != nil {
		return "", err
	}
	return getUserTextInput(prompt)
}

func (r *SystemKit) GetUserChoiceInput(ctx context.Context, vars *Vars, name string, args map[string]any) (string, error) {
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
