package agent

import (
	"os"

	"github.com/spf13/cobra"
)

func addAgentFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	// --agent agent/command or @agent/command
	flags.StringP("agent", "a", "", "Specify the agent to use. @<agent>")

	//
	flags.String("editor", "", "Specify the editor to use. default: builtin")
	flags.BoolP("edit", "e", false, "Launch editor")

	flags.MarkHidden("editor")

	// mainly when stdin is not desirable or possible
	// e.g. for testing or in vscode debug mode
	// prepend to other types of input
	flags.String("message", "", "Specify input message. Skip stdin")
	flags.MarkHidden("message")

	// // TODO further research: user role instruction/tool calls seem to work better and are preferred
	// // flags.VarP(newFilesValue([]string{}, &internal.InputFiles), "file", "", `Read file inputs.  May be given multiple times.`)
	// flags.StringArray("file", nil, `Read file inputs.  May be given multiple times.`)
	// flags.MarkHidden("file")

	// // doc agent
	// // flags.VarP(newTemplateValue("", &internal.TemplateFile), "template", "", "Document template file")
	// flags.String("template", "", "Document template file")
	// flags.MarkHidden("template")

	// use flags in case when special chars do not work
	flags.Bool("stdin", false, "Read input from stdin. '-'")

	flags.Bool("pb-read", false, "Read input from clipboard. '{'")
	flags.Bool("pb-tail", false, "Read input from clipboard and wait. '{{'")

	// // special inputs
	// flags.Bool("screenshot", false, "Take screenshot of the active tab in Chrome (CRX)")
	// flags.Bool("voice", false, "Transcribe voice using speech recognition in Chrome (CRX)")

	// flags.MarkHidden("screenshot")
	// flags.MarkHidden("voice")

	// output
	// flags.StringVar(&internal.OutputFlag, "output", "", "Save final response to a file.")
	flags.String("output", "", "Save final response to a file.")

	// use flags
	flags.Bool("pb-write", false, "Copy output to clipboard. '}'")
	flags.Bool("pb-append", false, "Append output to clipboard. '}}'")

	// flags.Var(newOutputValue("markdown", &internal.FormatFlag), "format", "Output format: raw, text, json, markdown, or tts.")
	flags.String("format", "markdown", "Output format: raw, text, json, markdown, or tts.")

	// security
	flags.String("deny", "rm,sudo", "List of comma separated system commands disallowed for tool calls. Approval is required to proceed. Ignored if 'unsafe' is true")
	flags.String("allow", "", "List of comma separated system commands allowed for tool calls")
	flags.Bool("unsafe", false, "Skip command security check to allow unsafe operations. Use with caution")

	flags.MarkHidden("deny")
	flags.MarkHidden("allow")

	// history
	// TODO
	// auto adjust based on relevance of messages to the current query
	// embedding/rag
	// summerization
	flags.BoolP("new", "n", false, "Start a new conversation")
	flags.String("chat", "", "Continue conversation with the chat id")

	flags.Int("max-history", 3, "Max number of historic messages")
	flags.Int("max-span", 480, "How far in minutes to go back in time for historic messages")

	flags.MarkHidden("max-history")
	flags.MarkHidden("max-span")

	//
	flags.Int("max-turns", 0, "Max number of turns")
	flags.Int("max-time", 0, "Max number of seconds for timeout")

	flags.MarkHidden("max-turns")
	flags.MarkHidden("max-time")

	// // mcp
	// flags.String("mcp-server-root", "", "MCP server config base directory")
	// flags.MarkHidden("mcp-server-root")

	// resource
	flags.String("resource", "resource.json", "Resource configuration")
	flags.MarkHidden("resource")

	// LLM
	// a set of models grouped under one name for convenience from potentially different service providers
	flags.StringP("models", "m", "", "LLM model alias defined in the models directory")

	flags.String("image-viewer", "", "Image viewer")

	flags.MarkHidden("image-viewer")

	//
	flags.String("log-level", "", "Set log level.")
	flags.MarkHidden("log-level")
	flags.Bool("quiet", false, "Operate quietly. Only show final response")
	flags.Bool("verbose", false, "Show progress and debugging information")
	flags.Bool("trace", false, "Turn on tracing")

	//
	flags.Bool("internal", false, "Enable internal agents and tools")
	flags.MarkHidden("internal")

	//
	flags.Bool("dry-run", false, "Enable dry run mode. No API call will be made")
	flags.String("dry-run-content", "", "Content returned for dry run")
	flags.MarkHidden("dry-run")
	flags.MarkHidden("dry-run-content")

	//
	flags.BoolP("interactive", "i", false, "Interactive mode")
	flags.String("shell", os.Getenv("SHELL"), "Shell to use for interactive mode")

	flags.MarkHidden("shell")

	// better tool call support instead?
	flags.StringP("workspace", "w", "", "Workspace directory")

	flags.MarkHidden("workspace")

	// TODO
	flags.Bool("watch", false, "Watch the workspace directory and respond to embedded ai requests in files")
	flags.Bool("pb-watch", false, "Watch system clipboard and respond to embedded ai requests. Copy output to clipboard")

	flags.MarkHidden("watch")
	flags.MarkHidden("pb-watch")

	// // mcp - this is for mcp, but we need to define it here
	// flags.Int("port", 0, "Port to run the server")
	// flags.String("host", "localhost", "Host to bind the server")
	// flags.MarkHidden("port")
	// flags.MarkHidden("host")
}
