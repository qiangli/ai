package agent

import (
	"os"

	"github.com/spf13/cobra"
)

func addAgentFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	// --agent agent/command or @agent/command
	flags.StringP("agent", "a", "ask", "Specify the agent to use. @<agent>")

	//
	flags.String("editor", "", "Specify the editor to use. default: builtin")
	flags.BoolP("edit", "e", false, "Launch editor")

	flags.MarkHidden("editor")

	// mainly when stdin is not desirable or possible
	// e.g. for testing or in vscode debug mode
	flags.String("message", "", "Specify input message. Skip stdin")
	flags.MarkHidden("message")

	// TODO further research: user role instruction/tool calls seem to work better and are preferred
	// flags.VarP(newFilesValue([]string{}, &internal.InputFiles), "file", "", `Read file inputs.  May be given multiple times.`)
	flags.StringArray("file", nil, `Read file inputs.  May be given multiple times.`)
	flags.MarkHidden("file")

	// doc agent
	// flags.VarP(newTemplateValue("", &internal.TemplateFile), "template", "", "Document template file")
	flags.String("template", "", "Document template file")
	flags.MarkHidden("template")

	// use flags in case when special chars do not work
	flags.Bool("stdin", false, "Read input from stdin. '-'")

	flags.Bool("pb-read", false, "Read input from clipboard. '{'")
	flags.Bool("pb-tail", false, "Read input from clipboard and wait. '{{'")

	// special inputs
	flags.Bool("screenshot", false, "Take screenshot of the active tab in Chrome (CRX)")
	flags.Bool("voice", false, "Transcribe voice using speech recognition in Chrome (CRX)")

	flags.MarkHidden("screenshot")
	flags.MarkHidden("voice")

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
	flags.Int("max-turns", 16, "Max number of turns")
	flags.Int("max-time", 3600, "Max number of seconds for timeout")

	flags.MarkHidden("max-time")

	// mcp
	flags.String("mcp-server-root", "", "MCP server config base directory")

	flags.MarkHidden("mcp-server-root")

	// resource
	flags.String("resource", "resource.json", "Resource configuration")
	flags.MarkHidden("resource")

	// LLM
	// a set of models grouped under one name for convenience from potentially different service providers
	flags.StringP("models", "m", "", "LLM model alias defined in the models directory")

	// // llm - use models config yaml instead
	// flags.String("provider", "", "LLM provider")
	// flags.MarkHidden("provider")

	// flags.String("api-key", "", "LLM API key")
	// flags.String("model", "", "LLM default model")
	// flags.String("base-url", "", "LLM Base URL")

	// flags.MarkHidden("api-key")
	// flags.MarkHidden("model")
	// flags.MarkHidden("base-url")

	// // basic/regular/reasoning models
	// flags.String("l1-api-key", "", "Level1 basic LLM API key")
	// flags.String("l1-model", "", "Level1 basic LLM model")
	// flags.String("l1-base-url", "", "Level1 basic LLM Base URL")

	// flags.String("l2-api-key", "", "Level2 standard LLM API key")
	// flags.String("l2-model", "", "Level2 standard LLM model")
	// flags.String("l2-base-url", "", "Level2 standard LLM Base URL")

	// flags.String("l3-api-key", "", "Level3 advanced LLM API key")
	// flags.String("l3-model", "", "Level3 advanced LLM model")
	// flags.String("l3-base-url", "", "Level3 advanced LLM Base URL")

	// flags.MarkHidden("l1-api-key")
	// flags.MarkHidden("l2-api-key")
	// flags.MarkHidden("l3-api-key")
	// flags.MarkHidden("l1-model")
	// flags.MarkHidden("l2-model")
	// flags.MarkHidden("l3-model")
	// flags.MarkHidden("l1-base-url")
	// flags.MarkHidden("l2-base-url")
	// flags.MarkHidden("l3-base-url")

	// tts
	// flags.String("tts-provider", "", "TTS provider")
	// flags.String("tts-api-key", "", "TTS API key")
	// flags.String("tts-model", "", "TTS model")
	// flags.String("tts-base-url", "", "TTS Base URL")

	// flags.MarkHidden("tts-provider")
	// flags.MarkHidden("tts-api-key")
	// flags.MarkHidden("tts-model")
	// flags.MarkHidden("tts-base-url")

	// // image -- use models config yaml
	// flags.String("image-api-key", "", "Image LLM API key")
	// flags.String("image-model", "", "Image LLM model")
	// flags.String("image-base-url", "", "Image LLM Base URL")

	// flags.MarkHidden("image-model")
	// flags.MarkHidden("image-api-key")
	// flags.MarkHidden("image-base-url")

	flags.String("image-viewer", "", "Image viewer")

	flags.MarkHidden("image-viewer")

	//
	flags.String("log", "", "Log all debugging information to a file")
	flags.Bool("quiet", false, "Operate quietly. Only show final response")
	flags.Bool("verbose", false, "Show progress and debugging information")
	flags.Bool("trace", false, "Turn on tracing")

	//
	flags.Bool("internal", false, "Enable internal agents and tools")

	flags.MarkHidden("internal")

	//
	// flags.String("role", "system", "Specify a role for the prompt")
	// flags.String("prompt", "", "Specify context instruction")
	// flags.MarkHidden("role")
	// flags.MarkHidden("prompt")

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

	// TODO move to individual config file
	// agent specific flags
	// db
	// flags.String("sql-db-host", "", "Database host")
	// flags.String("sql-db-port", "", "Database port")
	// flags.String("sql-db-username", "", "Database username")
	// flags.String("sql-db-password", "", "Database password")
	// flags.String("sql-db-name", "", "Database name")

	// flags.MarkHidden("sql-db-host")
	// flags.MarkHidden("sql-db-port")
	// flags.MarkHidden("sql-db-username")
	// flags.MarkHidden("sql-db-password")
	// flags.MarkHidden("sql-db-name")

	// mcp - this is for mcp, but we need to define it here
	flags.Int("port", 0, "Port to run the server")
	flags.String("host", "localhost", "Host to bind the server")

	flags.MarkHidden("port")
	flags.MarkHidden("host")
}
