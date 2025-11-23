package conf

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"

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

// ParseArgs returns nil and no error for non agent tool commands
func ParseActionArgs(args []string) (*api.ActionConfig, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing command args")
	}
	// skip trigger word "ai"
	if len(args) > 0 && strings.ToLower(args[0]) == "ai" {
		args = args[1:]
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("empty ai command args")
	}

	var name = args[0]
	args = args[1:]

	var kit string
	var ftype string
	switch name[0] {
	case '@':
		name = strings.ToLower(name[1:])
		ftype = api.ToolTypeAgent
		kit = api.ToolTypeAgent
	case '/':
		name = strings.ToLower(name[1:])
		kit, name = api.KitName(name).Decode()
	default:
		if strings.HasPrefix(name, "agent:") {
			name = strings.ToLower(name[6:])
			ftype = api.ToolTypeAgent
			kit = api.ToolTypeAgent
		} else if strings.HasSuffix(name, ",") {
			name = strings.ToLower(name[:len(name)-1])
			ftype = api.ToolTypeAgent
			kit = api.ToolTypeAgent
		} else {
			// not an agent/tool command
			return nil, nil
		}
	}
	if name == "" {
		name = "anonymous"
		ftype = api.ToolTypeAgent
		kit = api.ToolTypeAgent
	}

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
	logLevel := fs.String("log-level", "", "Log level: quiet, info, verbose, trace")

	//
	err := fs.Parse(args)
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
		for _, v := range args {
			if v == "--"+fl || v == "-"+fl || strings.HasPrefix(v, "--"+fl+"=") || strings.HasPrefix(v, "-"+fl+"=") {
				return true
			}
		}
		return false
	}

	// agent/tool default arguments
	// precedence: <common>, arg slice, arguments
	var atArgs = make(map[string]any)
	// Parse JSON arguments
	if *arguments != "" {
		if err := json.Unmarshal([]byte(*arguments), &atArgs); err != nil {
			return nil, fmt.Errorf("invalid JSON arguments: %w", err)
		}
	}
	// Parse individual arg in the slice
	for _, v := range arg {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			atArgs[parts[0]] = parts[1]
		} else {
			return nil, fmt.Errorf("invalid argument format: %s", v)
		}
	}
	// common flags
	for k, v := range common {
		if isSet(k) {
			atArgs[k] = v
		}
	}

	// update the map
	if kit != "" {
		atArgs["kit"] = kit
	}
	if name != "" {
		atArgs["name"] = name
	}
	if msg != "" {
		atArgs["message"] = msg
	}
	if prompt != "" {
		atArgs["instruction"] = prompt
	}
	if *model != "" {
		atArgs["model"] = *model
	}

	var at = &api.ActionConfig{
		Message:     msg,
		Instruction: prompt,
		Arguments:   atArgs,
		Kit:         kit,
		Name:        name,
		Type:        ftype,
	}

	return at, nil
}

// agent/tool name is "ai" or starts with "agent:", "@" or "/" or ends with ","
// ai args...
//
// agent:name args...
// @name args...
// /name args...
//
// anoymous:
// ai args...
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
