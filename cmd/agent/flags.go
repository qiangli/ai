package agent

import (
	"os"

	"github.com/spf13/cobra"
)

func addAgentFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	// --agent agent/command or @agent/command
	flags.StringP("agent", "a", "", "Specify the agent to use as @<agent>")

	// Mainly when stdin is not desirable or possible
	// e.g., for testing or in VSCode debug mode
	flags.String("message", "", "Input message")

	flags.String("chat", "", "Continue conversation using chat ID")

	flags.BoolP("new", "n", false, "Start a new conversation")
	flags.Int("max-history", 3, "Max historic messages to retrieve")
	flags.Int("max-span", 480, "Historic message retrieval span (minutes)")
	flags.String("context", "", "Agent for summarizing history")
	flags.Int("max-turns", 0, "Max conversation turns")
	flags.Int("max-time", 0, "Max timeout (seconds)")

	flags.String("format", "markdown", "Output as raw, text, json, or markdown")

	flags.Bool("unsafe", false, "Allow unsafe operations (skip security check)")
	flags.StringP("workspace", "w", "", "Workspace directory path")

	flags.String("log-level", "", "Log level: quiet, info, verbose, trace")

	flags.Bool("quiet", false, "Operate quietly. Only show final response")
	flags.Bool("info", false, "Show progress")
	flags.Bool("verbose", false, "Show progress and debugging information")
	flags.Bool("trace", false, "Turn on tracing")

	flags.MarkHidden("quiet")
	flags.MarkHidden("info")
	flags.MarkHidden("verbose")
	flags.MarkHidden("trace")

	// LLM
	// a set of models grouped under one name for convenience from potentially different service providers
	flags.StringP("models", "m", "", "LLM model alias defined in the models directory")
	flags.MarkHidden("models")

	//
	flags.Bool("dry-run", false, "Enable dry run mode. No API call will be made")
	flags.String("dry-run-content", "", "Content returned for dry run")

	flags.MarkHidden("dry-run")
	flags.MarkHidden("dry-run-content")

	//
	flags.String("editor", "", "Specify the editor to use. default: builtin")
	flags.BoolP("edit", "e", false, "Launch editor")

	flags.MarkHidden("editor")
	flags.MarkHidden("edit")

	// // TODO further research: user role instruction/tool calls seem to work better and are preferred
	// // flags.VarP(newFilesValue([]string{}, &internal.InputFiles), "file", "", `Read file inputs.  May be given multiple times.`)
	// flags.StringArray("file", nil, `Read file inputs.  May be given multiple times.`)
	// flags.MarkHidden("file")
	// // doc agent
	// // flags.VarP(newTemplateValue("", &internal.TemplateFile), "template", "", "Document template file")
	// flags.String("template", "", "Document template file")
	// flags.MarkHidden("template")
	// // special inputs
	// flags.Bool("screenshot", false, "Take screenshot of the active tab in Chrome (CRX)")
	// flags.Bool("voice", false, "Transcribe voice using speech recognition in Chrome (CRX)")
	// flags.MarkHidden("screenshot")
	// flags.MarkHidden("voice")

	// use flags in case when special chars do not work
	flags.Bool("stdin", false, "Read input from stdin. '-'")
	flags.MarkHidden("stdin")

	// disable auto is-piped check/ignore stdin flags
	// this is for vscode debugging or in cases when reading stdin is not desiirable
	flags.Bool("no-stdin", false, "Disable reading input from stdin")
	flags.MarkHidden("no-stdin")

	flags.Bool("pb-read", false, "Read input from clipboard. '{'")
	flags.Bool("pb-tail", false, "Read input from clipboard and wait. '{{'")
	flags.MarkHidden("pb-read")
	flags.MarkHidden("pb-tail")

	flags.Bool("pb-write", false, "Copy output to clipboard. '}'")
	flags.Bool("pb-append", false, "Append output to clipboard. '}}'")
	flags.MarkHidden("pb-write")
	flags.MarkHidden("pb-append")

	// // output
	// // flags.StringVar(&internal.OutputFlag, "output", "", "Save final response to a file.")
	// flags.String("output", "", "Save final response to a file.")
	// flags.MarkHidden("output")

	// // resource
	// flags.String("resource", "resource.json", "Resource configuration")
	// flags.MarkHidden("resource")

	flags.String("image-viewer", "", "Image viewer")
	flags.MarkHidden("image-viewer")

	//
	flags.BoolP("interactive", "i", false, "Interactive mode")
	flags.String("shell", os.Getenv("SHELL"), "Shell to use for interactive mode")

	flags.MarkHidden("interactive")
	flags.MarkHidden("shell")

	// TODO
	flags.Bool("watch", false, "Watch the workspace directory and respond to embedded ai requests in files")
	flags.Bool("pb-watch", false, "Watch system clipboard and respond to embedded ai requests. Copy output to clipboard")

	flags.MarkHidden("watch")
	flags.MarkHidden("pb-watch")
}
