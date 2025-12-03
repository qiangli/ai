package conf

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

	"github.com/u-root/u-root/pkg/shlex"

	"github.com/qiangli/ai/swarm/api"
)

// Custom type for string array
type stringSlice []string

// String method to satisfy the flag.Value interface
func (s *stringSlice) String() string {
	return fmt.Sprint(*s)
}

// Set method to satisfy the flag.Value interface
func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

// ParseActionArgs parses and converts arguments list to map
func ParseActionArgs(argv []string) (api.ArgMap, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("missing command arguments")
	}
	// skip trigger word "ai"
	if len(argv) > 0 && strings.ToLower(argv[0]) == "ai" {
		argv = argv[1:]
	}
	if len(argv) == 0 {
		return nil, fmt.Errorf("empty ai command arguments")
	}

	var name = argv[0]

	var kit string
	// var ftype string
	switch name[0] {
	case '@':
		name = strings.ToLower(name[1:])
		// ftype = api.ToolTypeAgent
		kit = api.ToolTypeAgent
		argv = argv[1:]
	case '/':
		name = strings.ToLower(name[1:])
		kit, name = api.Kitname(name).Decode()
		argv = argv[1:]
	default:
		if strings.HasPrefix(name, "agent:") {
			name = strings.ToLower(name[6:])
			// ftype = api.ToolTypeAgent
			kit = api.ToolTypeAgent
			argv = argv[1:]
		} else if strings.HasSuffix(name, ",") {
			name = strings.ToLower(name[:len(name)-1])
			// ftype = api.ToolTypeAgent
			kit = api.ToolTypeAgent
			argv = argv[1:]
		} else {
			name = ""
			kit = api.ToolTypeAgent

			// not an agent/tool command
			// return nil, nil
			// assume all are agent/tool
			// use IsAgentTool for checking if needed
		}
	}
	// if name == "" {
	// 	// name = "anonymous"
	// 	// ftype = api.ToolTypeAgent
	// 	kit = api.ToolTypeAgent
	// }

	fs := flag.NewFlagSet("ai", flag.ContinueOnError)

	var arg stringSlice
	fs.Var(&arg, "arg", "argument name=value (can be used multiple times)")
	arguments := fs.String("arguments", "", "arguments map in JSON format")

	//
	instruction := fs.String("instruction", "", "System role prompt message")
	message := fs.String("message", "", "User input message")
	//
	model := fs.String("model", "", "LLM model alias defined in the model set")

	// common args
	maxHistory := fs.Int("max-history", 0, "Max historic messages to retrieve")
	maxSpan := fs.Int("max-span", 0, "Historic message retrieval span (minutes)")
	maxTurns := fs.Int("max-turns", 3, "Max conversation turns")
	maxTime := fs.Int("max-time", 30, "Max timeout (seconds)")
	format := fs.String("format", "json", "Output as text or json")

	workspace := fs.String("workspace", "", "Workspace root path")

	logLevel := fs.String("log-level", "", "Log level: quiet, info, verbose, trace")
	isQuiet := fs.Bool("quiet", false, "Operate quietly, only show final response. log-level=quiet")
	isInfo := fs.Bool("info", false, "Show progress")
	isVerbose := fs.Bool("verbose", false, "Show progress and debugging information")

	//
	err := fs.Parse(argv)
	if err != nil {
		return nil, err
	}

	var common = map[string]any{
		"max-history": *maxHistory,
		"max-span":    *maxSpan,
		"max-turns":   *maxTurns,
		"max-time":    *maxTime,
		"format":      *format,
		"log-level":   *logLevel,
	}

	// prepend messsage to non flag/option args
	var msg = strings.Join(fs.Args(), " ")
	if *message != "" {
		msg = *message + " " + msg
	}
	msg = strings.TrimSpace(msg)

	prompt := strings.TrimSpace(*instruction)

	//
	isSet := func(fl string) bool {
		fl = strings.ToLower(fl)
		for _, v := range argv {
			if v == "--"+fl || v == "-"+fl || strings.HasPrefix(v, "--"+fl+"=") || strings.HasPrefix(v, "-"+fl+"=") {
				return true
			}
		}
		return false
	}

	// agent/tool default arguments
	// precedence: <common>, arg slice, arguments
	var argm = make(map[string]any)
	// Parse JSON arguments
	if *arguments != "" {
		if err := json.Unmarshal([]byte(*arguments), &argm); err != nil {
			return nil, fmt.Errorf("invalid JSON arguments: %w", err)
		}
	}
	// Parse individual arg in the slice
	for _, v := range arg {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			argm[parts[0]] = parts[1]
		} else {
			return nil, fmt.Errorf("invalid argument format: %s", v)
		}
	}
	// common flags
	for k, v := range common {
		if isSet(k) {
			argm[k] = v
		}
	}

	//
	if *isVerbose {
		argm["log-level"] = "verbose"
	}
	if *isInfo {
		argm["log-level"] = "info"
	}
	if *isQuiet {
		argm["log-level"] = "quiet"
	}

	// update the map
	argm["workspace"] = *workspace

	if kit != "" {
		argm["kit"] = kit
	}
	if name != "" {
		argm["name"] = name
	}
	if msg != "" {
		argm["message"] = msg
	}
	if prompt != "" {
		argm["instruction"] = prompt
	}
	if *model != "" {
		argm["model"] = *model
	}

	// var at = &api.ActionConfig{
	// 	Message:     msg,
	// 	Instruction: prompt,
	// 	Arguments:   atArgs,
	// 	Kit:         kit,
	// 	Name:        name,
	// 	Type:        ftype,
	// }

	return argm, nil
}

// agent/tool name is "ai" or starts with "agent:", "@" or "/" or ends with ","
// ai name args...
//
// agent:name args...
// @name args...
// /name args...
//
// anoymous:
// @ args...
// / args...
//
// name,
func IsAgentTool(s string) bool {
	if len(s) == 0 {
		return false
	}
	switch s[0] {
	case '@':
		return true
	case '/':
		return true
	default:
		if strings.HasPrefix(s, "agent:") {
			return true
		}
	}
	s, _ = split2(s, " ", "")
	if s == "ai" {
		return true
	}
	if strings.HasSuffix(s, ",") {
		return true
	}
	return false
}

func ParseActionCommand(s string) (api.ArgMap, error) {
	argv := shlex.Argv(s)
	at, err := ParseActionArgs(argv)
	if err != nil {
		return nil, err
	}
	if at == nil {
		return nil, fmt.Errorf("invalid command: %v", s)
	}
	return at, nil
}
