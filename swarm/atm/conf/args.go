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
// agent tool name must start with @ or /
// ai:
// ai @name args...
// ai /name args...
//
// @name args...
// /name args...
//
// anoymous:
// @ args...
// / args...
func ParseArgs(args []string) (*api.AgentTool, error) {
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
	case '/':
		name = strings.ToLower(name[1:])
		if strings.HasPrefix(name, "agent:") {
			ftype = api.ToolTypeAgent
		}
		kit, _ = api.KitName(name[1:]).Decode()
	default:
		// not an agent/tool command
		return nil, nil
	}
	if name == "" {
		name = "anonymous"
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
	atArgs["name"] = name
	atArgs["message"] = msg
	atArgs["instruction"] = prompt
	atArgs["model"] = *model

	var at = &api.AgentTool{
		Message:     msg,
		Instruction: prompt,
		Arguments:   atArgs,
		Kit:         kit,
		Name:        name,
		Type:        ftype,
	}

	return at, nil
}
