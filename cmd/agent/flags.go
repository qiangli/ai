package agent

import (
	"github.com/spf13/cobra"
)

func addAgentFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.String("arguments", "", "arguments map in JSON format")
	flags.StringArray("arg", []string{}, "argument name=value (can be used multiple times)")

	flags.String("agent", "", "Specify the agent to use. shorthand: @<agent>")
	flags.String("instruction", "", "System role prompt")
	flags.String("message", "", "User query")

	// flags.Bool("new", false, "Start a new conversation. max-history=0 and max-span=0")
	flags.String("context", "", "Agent for summarizing history")
	// flags.MarkHidden("new")
	flags.MarkHidden("context")

	flags.Int("max-history", 3, "Max historic messages to retrieve")
	flags.Int("max-span", 480, "Historic message retrieval span (minutes)")
	flags.Int("max-turns", 0, "Max conversation turns")
	flags.Int("max-time", 0, "Max timeout (seconds)")

	flags.String("format", "markdown", "Output as raw, text, json, or markdown")

	flags.StringP("workspace", "w", "", "Workspace root path")

	flags.String("log-level", "info", "Log level: quiet, info, verbose, trace")

	flags.Bool("quiet", false, "Operate quietly, only show final response. log-level=quiet")
	flags.Bool("info", true, "Show progress")
	flags.Bool("verbose", false, "Show progress and debugging information")
	flags.Bool("trace", false, "Turn on tracing")

	flags.MarkHidden("quiet")
	flags.MarkHidden("info")
	flags.MarkHidden("verbose")
	flags.MarkHidden("trace")

	// LLM
	// a set of model grouped under one name for convenience from potentially different service providers
	flags.StringP("model", "m", "", "LLM model alias defined in the model set")

	// disable auto is-piped check/ignore stdin flags
	// this is for vscode debugging or in cases when reading stdin is not desiirable
	flags.Bool("no-stdin", false, "Disable reading input from stdin")
	flags.MarkHidden("no-stdin")
}
