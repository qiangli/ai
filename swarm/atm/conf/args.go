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
// skipping trigger word "ai"
func ParseActionArgs(argv []string) (api.ArgMap, error) {
	// argv = dropEmpty(argv)

	if len(argv) == 0 {
		return nil, fmt.Errorf("missing action arguments")
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
	switch name[0] {
	case '@':
		name = strings.ToLower(name[1:])
		kit = string(api.ToolTypeAgent)
		argv = argv[1:]
	case '/':
		name = strings.ToLower(name[1:])
		kit, name = api.Kitname(name).Decode()
		if kit == "" {
			// not a tool
			name = ""
		} else {
			argv = argv[1:]
		}
	default:
		if strings.HasPrefix(name, "agent:") {
			name = strings.ToLower(name[6:])
			kit = string(api.ToolTypeAgent)
			argv = argv[1:]
		} else if strings.HasSuffix(name, ",") {
			name = strings.ToLower(name[:len(name)-1])
			kit = string(api.ToolTypeAgent)
			argv = argv[1:]
		} else {
			name = ""
			// missing action (agent/tool)
			// use IsAgentTool for checking if needed
		}
	}

	fs := flag.NewFlagSet("ai", flag.ContinueOnError)

	var arg stringSlice
	fs.Var(&arg, "arg", "argument name=value (can be used multiple times)")
	// for LLM: json object format
	// for human: also support string of name=value delimited by space and array list of name=value in json format
	arguments := fs.String("arguments", "", "arguments in json object format")

	//
	instruction := fs.String("instruction", "", "System role prompt message")
	message := fs.String("message", "", "User input message")
	//
	model := fs.String("model", "", "LLM model alias defined in the model set")

	// common args
	// TODO revisit
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

	// tool
	command := fs.String("command", "", "Shell command(s) to be executed.")
	script := fs.String("script", "", "Path to the shell script file to be executed.")

	//
	err := fs.Parse(argv)
	if err != nil {
		return nil, err
	}

	var common = map[string]any{
		"max_history": *maxHistory,
		"max_span":    *maxSpan,
		"max_turns":   *maxTurns,
		"max_time":    *maxTime,
		"format":      *format,
		"log_level":   *logLevel,
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
	// Parse string arguments
	// var argm map[string]any
	var argm = make(map[string]any)
	if *arguments != "" {
		// if v, err := ParseArguments(*arguments); err != nil {
		// 	return nil, err
		// } else {
		// 	argm = v
		// }
		args := *arguments
		switch {
		case strings.HasPrefix(args, "{"):
			if err := json.Unmarshal([]byte(args), &argm); err != nil {
				return nil, fmt.Errorf("invalid json object arguments: %q error: %w", args, err)
			}
		case strings.HasPrefix(args, "["):
			var argv []string
			if err := json.Unmarshal([]byte(args), &argv); err != nil {
				return nil, fmt.Errorf("invalid json array arguments: %q error: %w", args, err)
			}
			argm["arguments"] = argv
			// if v, err := parse(argv); err != nil {
			// 	return nil, err
			// } else {
			// 	argm = v
			// }
		default:
			// string name=value pairs
			argv := shlex.Argv(args)
			// if v, err := parse(argv); err != nil {
			// 	return nil, err
			// } else {
			// 	argm = v
			// }
			argm["arguments"] = argv
		}
	}
	// if argm == nil {
	// 	argm = make(map[string]any)
	// }

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
		argm["log_level"] = "verbose"
	}
	if *isInfo {
		argm["log_level"] = "info"
	}
	if *isQuiet {
		argm["log_level"] = "quiet"
	}

	// update the map
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
	if *workspace != "" {
		argm["workspace"] = *workspace
	}
	if *command != "" {
		argm["command"] = *command
	}
	if *script != "" {
		argm["script"] = *script
	}

	return argm, nil
}

// IsAction returns true if string s is an action command.
// action (agent and tool) name convention:
// "ai" or prefix "agent:", "@" or "/" or suffix ","
//
// ai [action] message...
//
// action
//
//   - agent: prefix "agent:",  at sign "@" or suffix comma ","
//     agent:pack[/sub]
//     @pack[/sub]
//     pack[/sub],
//
//   - tool: slash command, prefix "/" followed by colon ":" or single component
//     /kit
//     /kit:name[/sub]
//     /agent:pack[/sub]
//
// anoymous:
// @ args...
// / args...
func IsAction(s string) bool {
	if len(s) == 0 {
		return false
	}
	switch s[0] {
	case '@':
		return true
	case '/':
		// assuming NO unix binary directly under root /.
		s = strings.TrimPrefix(s, "/")
		sa := strings.SplitN(s, "/", 2)
		if len(sa) == 1 {
			return true
		}
		return strings.Contains(s, ":")
	default:
		if strings.HasPrefix(s, "agent:") {
			return true
		}
	}
	s, _ = split2(s, " ", "")
	// true but empty command
	if s == "ai" {
		return true
	}
	if strings.HasSuffix(s, ",") {
		return true
	}
	return false
}

func ParseActionCommand(s string) (api.ArgMap, error) {
	if len(s) == 0 {
		return nil, fmt.Errorf("missing action command")
	}
	argv := shlex.Argv(s)
	argm, err := ParseActionArgs(argv)
	if err != nil {
		return nil, err
	}
	if len(argm) == 0 {
		return nil, fmt.Errorf("empty command: %v", s)
	}
	return argm, nil
}

// Parse arguments in various forms.
// + json object
// + array list of name=value pairs or command line style options
// + string of name=value pairs or command line style options
// func ParseArguments(args string) (map[string]any, error) {
// 	if len(args) == 0 {
// 		return nil, nil
// 	}
// 	// if any of the args starts with "-", an invalid name,
// 	// assume command line style options
// 	// otherise, treat as name=value pairs
// 	isCmdline := func(a []string) bool {
// 		for _, v := range a {
// 			if strings.HasPrefix(v, "-") {
// 				return true
// 			}
// 		}
// 		return false
// 	}
// 	parse := func(a []string) (map[string]any, error) {
// 		if isCmdline(a) {
// 			return ParseActionArgs(a)
// 		}
// 		var m = make(map[string]any)
// 		for _, v := range a {
// 			s2 := strings.SplitN(v, "=", 2)
// 			if len(s2) == 2 {
// 				m[s2[0]] = s2[1]
// 			}
// 		}
// 		return m, nil
// 	}

// 	// Parse arguments
// 	var argm map[string]any
// 	switch {
// 	case strings.HasPrefix(args, "{"):
// 		if err := json.Unmarshal([]byte(args), &argm); err != nil {
// 			return nil, fmt.Errorf("invalid json object arguments: %q error: %w", args, err)
// 		}
// 	case strings.HasPrefix(args, "["):
// 		var argv []string
// 		if err := json.Unmarshal([]byte(args), &argv); err != nil {
// 			return nil, fmt.Errorf("invalid json array arguments: %q error: %w", args, err)
// 		}
// 		if v, err := parse(argv); err != nil {
// 			return nil, err
// 		} else {
// 			argm = v
// 		}
// 	default:
// 		// string name=value pairs
// 		argv := shlex.Argv(args)
// 		if v, err := parse(argv); err != nil {
// 			return nil, err
// 		} else {
// 			argm = v
// 		}
// 	}

// 	return argm, nil
// }

// // remove empty items
// func dropEmpty(argv []string) []string {
// 	var mod []string
// 	for _, v := range argv {
// 		if len(v) > 0 {
// 			mod = append(mod, v)
// 		}
// 	}
// 	return mod
// }
