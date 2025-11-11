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

// name:
// ai @name args...
// ai /name args...
// @name args...
// /name args...
//
// anoymous:
// @ args...
// / args...
func ParseAgentToolArgs(owner string, args []string) (*api.AgentTool, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing args")
	}
	// ignore trigger word "ai"
	if len(args) > 0 && strings.ToLower(args[0]) == "ai" {
		args = args[1:]
	}
	if len(args) == 0 {
		return nil, fmt.Errorf("empty command line")
	}
	var name = strings.ToLower(args[0])
	args = args[1:]

	//
	fs := flag.NewFlagSet("ai", flag.ContinueOnError)
	arguments := fs.String("arguments", "", "arguments map in JSON format")
	var arg stringSlice
	fs.Var(&arg, "arg", "argument name=value (can be used multiple times)")

	//
	instruction := fs.String("instruction", "", "System role prompt message")
	message := fs.String("message", "", "User input message")
	//

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

	// better way?
	isSet := func(fl string) bool {
		for _, v := range args {
			if v == "--"+fl || v == "-"+fl || strings.HasPrefix(v, "--"+fl+"=") || strings.HasPrefix(v, "-"+fl+"=") {
				return true
			}
		}
		return false
	}

	// agent/tool default arguments
	var atArgs = make(map[string]any)
	// Parse JSON arguments
	if *arguments != "" {
		if err := json.Unmarshal([]byte(*arguments), &atArgs); err != nil {
			return nil, fmt.Errorf("invalid JSON arguments: %w", err)
		}
	}

	// Parse individual args
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

	//
	atArgs["name"] = name
	atArgs["message"] = msg
	atArgs["instruction"] = prompt

	newAgent := func(s string) *api.Agent {
		return &api.Agent{
			Owner: owner,
			Name:  s,
			//
			Adapter: "",
			Model:   nil,
			Tools:   nil,
		}
	}

	newTool := func(s string) *api.ToolFunc {
		kit, name := api.KitName(s).Decode()
		return &api.ToolFunc{
			Kit:       kit,
			Name:      name,
			Arguments: atArgs,
			// required fields need to be set later
			Type:       "",
			Parameters: nil,
		}
	}

	var at = &api.AgentTool{
		Owner:       owner,
		Message:     msg,
		Instruction: prompt,
		Arguments:   atArgs,
	}

	switch name[0] {
	case '@':
		at.Agent = newAgent(name[1:])
	case '/':
		at.Tool = newTool(name[1:])
	default:
		// unreachable
		// system/builtin bash command
	}

	return at, nil
}
