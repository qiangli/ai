package conf

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/u-root/u-root/pkg/shlex"

	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/flag"
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

// Parse and convert argument array list to map
// skipping trigger word "ai"
// for option args: replace dash "-" with understcore "_" in argument names
func ParseActionArgs(argv []string) (api.ArgMap, error) {
	if len(argv) == 0 {
		return map[string]any{}, nil
	}
	// skip trigger word "ai"
	if len(argv) > 0 && strings.ToLower(argv[0]) == "ai" {
		argv = argv[1:]
	}
	if len(argv) == 0 {
		return map[string]any{}, nil
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
			// not a tool, assuming system command
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
		}
	}

	fs := flag.NewFlagSet("ai", flag.ContinueOnError)

	var option stringSlice
	fs.Var(&option, "option", "argument name=value pair (can be used multiple times)")
	// for LLM: json object format
	// for human: support string of name=value delimited by space and array list of name=value in json format
	arguments := fs.String("arguments", "", "arguments in json object format")

	// LLM prompt/query/model
	instruction := fs.String("instruction", "", "System role prompt message")
	message := fs.String("message", "", "User input message")

	model := fs.String("model", "", "LLM model alias defined in the model set")

	// common args with defaut value
	maxHistory := fs.Int("max-history", api.DefaultMaxHistory, "Max number of historic messages to retrieve")
	maxSpan := fs.Int("max-span", api.DefaultMaxSpan, "Historic message retrieval span (minutes)")
	maxTurns := fs.Int("max-turns", api.DefaultMaxTurns, "Max conversation turns")
	maxTime := fs.Int("max-time", api.DefaultMaxTime, "Max timeout (seconds)")

	// format := fs.String("format", "json", "Output as text, json, or markdown")

	// logging
	logLevel := fs.String("log-level", "quiet", "Log level: quiet, info, verbose, trace")
	isQuiet := fs.Bool("quiet", false, "Operate quietly, only show final response. log-level=quiet")
	isInfo := fs.Bool("info", false, "Show progress")
	isVerbose := fs.Bool("verbose", false, "Show progress and debugging information")

	// special input
	// value provided as option
	stdin := fs.String("stdin", "", "Read input from stdin")

	//
	argm, err := fs.Parse(argv)
	if err != nil {
		return nil, err
	}

	var common = map[string]any{
		"max_history": *maxHistory,
		"max_span":    *maxSpan,
		"max_turns":   *maxTurns,
		"max_time":    *maxTime,
		"log_level":   *logLevel,
	}

	// prepend messsage to non flag/option args
	var msg = strings.Join(fs.Args(), " ")
	if *message != "" {
		msg = *message + " " + msg
	}
	msg = strings.TrimSpace(msg)

	instructStr := strings.TrimSpace(*instruction)
	modelStr := strings.TrimSpace(*model)

	//
	isSet := func(fl string) bool {
		fl = strings.ToLower(fl)
		fl = strings.ReplaceAll(fl, "_", "-")
		for _, v := range argv {
			if v == "--"+fl || v == "-"+fl || strings.HasPrefix(v, "--"+fl+"=") || strings.HasPrefix(v, "-"+fl+"=") {
				return true
			}
		}
		return false
	}

	// action agent/tool/command default arguments
	// precedence: <common>, arg slice, arguments
	// Parse string arguments
	// var argm map[string]any
	// var argm = make(map[string]any)
	if *arguments != "" {
		args := *arguments
		switch {
		case strings.HasPrefix(args, "{"):
			if err := json.Unmarshal([]byte(args), &argm); err != nil {
				return nil, fmt.Errorf("invalid json object arguments: %q error: %w", args, err)
			}
		case strings.HasPrefix(args, "["):
			var argv []string
			if err := json.Unmarshal([]byte(args), &argv); err != nil {
				// try parsing as comma separated list
				argv = api.ParseStringArray(args)
				if len(argv) == 0 {
					return nil, fmt.Errorf("invalid json array arguments: %q error: %w", args, err)
				}
			}
			argm["arguments"] = argv
		default:
			argv := shlex.Argv(args)
			argm["arguments"] = argv
		}
	}

	// Parse individual arg in the slice
	for _, v := range option {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			key := parts[0]
			key = strings.ReplaceAll(key, "-", "_")
			argm[key] = parts[1]
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
	if kit == "agent" {
		pack, sub := api.Packname(name).Decode()
		argm["name"] = sub
		argm["pack"] = pack
	}
	if msg != "" {
		argm["message"] = msg
	}
	if instructStr != "" {
		argm["instruction"] = instructStr
	}
	if modelStr != "" {
		argm["model"] = modelStr
	}

	if *stdin != "" {
		argm["stdin"] = *stdin
	}

	// replace all "-" with "_"
	for k, v := range argm {
		if strings.ContainsAny(k, "-") {
			key := strings.ReplaceAll(k, "-", "_")
			delete(argm, k)
			argm[key] = v
		}
	}

	return argm, nil
}

// Return true if command line "s" is an action command.
// action (agent, tool, and command) name convention:
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
		// return IsSlashTool(s)
		return true
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

// // Split command line  into array of words, parse and return the arguments map
// func ParseActionCommand(s string) (api.ArgMap, error) {
// 	if len(s) == 0 {
// 		return nil, fmt.Errorf("missing action command")
// 	}
// 	argv := shlex.Argv(s)
// 	argm, err := ParseActionArgs(argv)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return argm, nil
// }

// Splits command line
func Argv(s string) []string {
	argv := shlex.Argv(s)
	return argv
}

// Return a map[string]any after parsing string, argument list
// and validating kit:name
func Parse(input any) (api.ArgMap, error) {
	if input == nil {
		return nil, fmt.Errorf("input is nil")
	}
	var argm map[string]any
	var err error

	switch input := input.(type) {
	case string:
		argv := Argv(input)
		argm, err = parsev(argv)
	case []string:
		argm, err = parsev(input)
	case map[string]any:
		argm = input
	case api.ArgMap:
		argm = input
	default:
		return nil, fmt.Errorf("input data type not supported: %v. supported data type: string, []string, map[string]any", input)
	}

	if err != nil {
		return nil, err
	}
	if len(argm) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	a := api.ArgMap(argm)
	kit, name := a.Kitname().Decode()
	// ensure name is set even if it is empty
	a["kit"] = kit
	a["name"] = name
	if kit == "agent" {
		pack, sub := api.Packname(name).Decode()
		a["pack"] = pack
		a["name"] = sub
	}
	return a, nil
}

func parsev(argv []string) (api.ArgMap, error) {
	argm, err := ParseActionArgs(argv)
	if err != nil {
		return nil, err
	}
	return argm, nil
}

// #!/bin ... [--script]
// #!/kit:name
// /kit:name
func parseCmdline(s string) string {
	if s == "" {
		return ""
	}
	sa := strings.SplitN(s, "\n", 2)
	cmdline := sa[0]
	cmdline = strings.TrimPrefix(cmdline, "#!")
	cmdline = strings.TrimSuffix(cmdline, "--script")
	cmdline = strings.TrimSpace(cmdline)

	ca := strings.SplitN(cmdline, " ", 2)
	// #!/bin/bash
	// remove system bash cmd
	if strings.HasSuffix(ca[0], "/sh") || strings.HasSuffix(ca[0], "/bash") {
		cmdline = ca[1]
	}
	return cmdline
	// // /agent:pack/sub
	// // /kit:name
	// if strings.Contains(sa[0], ":") {
	// 	return cmdline
	// }
	// // skip first arg
	// // expected:
	// // ai [action] or [action]
	// return ca[1]
}

// #!/bin/bash ... [--script]
// [content]
// Return nil if number of lines is less than 2.
func ParseScriptCmdline(mime, v string) map[string]any {
	if v == "" {
		return nil
	}
	var sa []string
	sa = strings.SplitN(v, "\n", 2)

	var args = map[string]any{
		"kit":  "sh",
		"name": "bash",
	}
	var script string
	if strings.HasPrefix(sa[0], "#!") {
		cmdline := parseCmdline(sa[0])
		cmdArgs, _ := Parse(cmdline)
		maps.Copy(args, cmdArgs)
		if len(sa) == 2 {
			script = strings.TrimSpace(sa[1])
		}
	} else {
		script = v
	}
	args["script"] = fmt.Sprintf("data:%s,%s", mime, script)

	return args
}
