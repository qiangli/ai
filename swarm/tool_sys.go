package swarm

import (
	"context"
	"fmt"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	// "os/exec"
	"reflect"
	// "sort"
	"strings"

	// "github.com/qiangli/ai/swarm/log"
	// utool "github.com/qiangli/ai/internal/tool"
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

// func callTplTool(ctx context.Context, vars *api.Vars, f *api.ToolFunc, args map[string]any) (string, error) {
// 	funcMap := map[string]any{
// 		"join":    strings.Join,
// 		"split":   strings.Split,
// 		"trim":    strings.TrimSpace,
// 		"default": Default,
// 		"spread":  Spread,
// 	}

// 	runCmd := func(cmd string, args ...string) (string, error) {
// 		result, err := execCommand(cmd, args, vars.Config.Debug)

// 		if err != nil {
// 			return result, err
// 		}
// 		if result == "" {
// 			return fmt.Sprintf("%s executed successfully", cmd), nil
// 		}
// 		return result, nil
// 	}

// 	// // Add system commands to the function map
// 	// for _, v := range vars.Config.ToolSystemCommands {
// 	// 	if _, err := exec.LookPath(v); err != nil {
// 	// 		log.GetLogger(ctx).Errorf("%s not found in PATH\n", v)
// 	// 		continue
// 	// 	}
// 	// 	funcMap[v] = func(args ...string) (string, error) {
// 	// 		return runCmd(v, args...)
// 	// 	}
// 	// }
// 	funcMap["exec"] = runCmd

// 	var body string
// 	var err error
// 	if f.Body != "" {
// 		body, err = applyTemplate(f.Body, args, funcMap)
// 		if err != nil {
// 			return "", err
// 		}
// 	}

// 	switch f.Type {
// 	case ToolTypeTemplate:
// 		return body, nil
// 	// case ToolTypeSql:
// 	// 	cred, err := dbCred(vars, args)
// 	// 	if err != nil {
// 	// 		return "", err
// 	// 	}
// 	// 	return sqlQuery(ctx, cred, body)
// 	case ToolTypeShell:
// 		cmdline := strings.TrimSpace(body)
// 		return execCommand(cmdline, nil, vars.Config.Debug)
// 	}

// 	return "", fmt.Errorf("unknown function type %s for tool %s", f.Type, f.Name)
// }

type LocalSystem struct {
	tool *SystemKit
}

func newLocalSystem() *LocalSystem {
	return &LocalSystem{
		tool: &SystemKit{},
	}
}

func (ls LocalSystem) Call(ctx context.Context, vars *api.Vars, f *api.ToolFunc, args map[string]any) (*api.Result, error) {
	return ls.callSystemTool(ctx, vars, f, args)
}

type SystemKit struct {
}

func (ls LocalSystem) callSystemTool(ctx context.Context, vars *api.Vars, f *api.ToolFunc, args map[string]any) (*api.Result, error) {
	// tool := &SystemKit{}
	callArgs := []any{ctx, vars, f.Name, args}
	v, err := CallKit(ls.tool, f.Config.Kit, f.Name, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to call system tool %s %s: %w", f.Config.Kit, f.Name, err)
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
