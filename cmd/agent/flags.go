package agent

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal"
)

const (
// DefaultLlmModel = "gpt-4.1-mini"
// L1LlmModel      = "gpt-4.1-mini"
// L2LlmModel      = "gpt-4.1"
// L3LlmModel      = "o4-mini"
// ImageLlmModel = "dall-e-3"
)

// Output format type
type outputValue string

func newOutputValue(val string, p *string) *outputValue {
	*p = val
	return (*outputValue)(p)
}
func (s *outputValue) Set(val string) error {
	// TODO json
	for _, v := range []string{"raw", "text", "json", "markdown", "tts"} {
		if val == v {
			*s = outputValue(val)
			return nil
		}
	}
	return fmt.Errorf("invalid output format: %v. supported: raw, markdown", val)
}
func (s *outputValue) Type() string {
	return "string"
}
func (s *outputValue) String() string { return string(*s) }

// Template type
type templateValue string

func newTemplateValue(val string, p *string) *templateValue {
	*p = val
	return (*templateValue)(p)
}
func (s *templateValue) Set(val string) error {
	matches, err := filepath.Glob(val)
	if err != nil {
		return errors.New("error during file globbing")
	}
	if len(matches) != 1 {
		return errors.New("exactly one file must be provided")
	}

	fileInfo, err := os.Stat(matches[0])
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return errors.New("a file is required")
	}

	*s = templateValue(matches[0])
	return nil
}

func (s *templateValue) Type() string {
	return "string"
}

func (s *templateValue) String() string { return string(*s) }

// Files type
type filesValue struct {
	value   *[]string
	changed bool
}

func newFilesValue(val []string, p *[]string) *filesValue {
	ssv := new(filesValue)
	ssv.value = p
	*ssv.value = val
	return ssv
}

func (s *filesValue) Set(val string) error {
	matches, err := filepath.Glob(val)
	if err != nil {
		return fmt.Errorf("error processing glob pattern: %w", err)
	}

	if matches == nil {
		// no matches ignore
		return nil
	}

	if !s.changed {
		*s.value = matches
		s.changed = true
	} else {
		*s.value = append(*s.value, matches...)
	}
	return nil
}
func (s *filesValue) Append(val string) error {
	*s.value = append(*s.value, val)
	return nil
}

func (s *filesValue) Replace(val []string) error {
	out := make([]string, len(val))
	for i, d := range val {
		var err error
		out[i] = d
		if err != nil {
			return err
		}
	}
	*s.value = out
	return nil
}

func (s *filesValue) GetSlice() []string {
	out := make([]string, len(*s.value))
	if s.value != nil {
		copy(out, *s.value)
	}
	return out
}

func (s *filesValue) Type() string {
	return "string"
}

func (s *filesValue) String() string {
	if len(*s.value) == 0 {
		return ""
	}
	str, _ := s.writeAsCSV(*s.value)
	return "[" + str + "]"
}

func (s *filesValue) writeAsCSV(vals []string) (string, error) {
	b := &bytes.Buffer{}
	w := csv.NewWriter(b)
	err := w.Write(vals)
	if err != nil {
		return "", err
	}
	w.Flush()
	return strings.TrimSuffix(b.String(), "\n"), nil
}

func addAgentFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	// --agent agent/command or @agent/command
	flags.StringP("agent", "a", "", "Specify the agent to use or @agent")

	//
	flags.String("editor", "", "Specify the editor to use. default: builtin")
	flags.BoolP("edit", "e", false, "Launch editor")

	// mainly when stdin is not desirable or possible
	// e.g. for testing or in vscode debug mode
	flags.String("message", "", "Specify input message. Skip stdin")
	// flags.String("content", "", "Specify input content. Skip stdin")

	flags.MarkHidden("message")
	// flags.MarkHidden("content")
	// flags.String("input", "", "Read input message from a file")

	flags.VarP(newFilesValue([]string{}, &internal.InputFiles), "file", "", `Read file inputs.  May be given multiple times.`)

	// TODO
	// for certain flags, auto add to user query?
	// image file is located at {{.image}}
	// .ai/prompts/
	// image.md
	// template.md
	// file.md
	// workspace.md
	// flags.String("image", "", "Path to input image file")

	flags.Bool("stdin", false, "Read input from stdin. '-'")

	flags.Bool("pb-read", false, "Read input from clipboard. '{'")
	flags.Bool("pb-tail", false, "Read input from clipboard and wait. '{{'")

	flags.MarkHidden("file")

	// special inputs
	flags.Bool("screenshot", false, "Take screenshot of the active tab in Chrome (CRX)")
	flags.Bool("voice", false, "Transcribe voice using speech recognition in Chrome (CRX)")

	// output
	flags.StringVar(&internal.OutputFlag, "output", "", "Save final response to a file.")
	flags.Bool("pb-write", false, "Copy output to clipboard. '}'")
	flags.Bool("pb-append", false, "Append output to clipboard. '}}'")

	flags.Var(newOutputValue("markdown", &internal.FormatFlag), "format", "Output format: raw, text, json, markdown, or tts.")

	// security
	flags.String("deny", "rm,sudo", "List of comma separated system commands disallowed for tool calls. Approval is required to proceed. Ignored if 'unsafe' is true")
	flags.String("allow", "", "List of comma separated system commands allowed for tool calls")
	flags.Bool("unsafe", false, "Skip command security check to allow unsafe operations. Use with caution")

	// history
	// TODO auto adjust based on relevance of messages to the current query
	flags.BoolP("new", "n", false, "Start a new conversation")
	flags.Int("max-history", 3, "Max number of historic messages")
	flags.Int("max-span", 480, "How far in minutes to go back in time for historic messages")

	//
	flags.MarkHidden("max-history")
	flags.MarkHidden("max-span")

	// mcp
	flags.String("mcp-server-root", "", "MCP server config base directory")

	flags.MarkHidden("mcp-server-root")

	// LLM
	flags.StringP("models", "m", "", "LLM model alias defined in the models directory")

	flags.String("provider", "", "LLM provider")
	flags.String("api-key", "", "LLM API key")
	flags.String("model", "", "LLM default model")
	flags.String("base-url", "", "LLM Base URL")

	flags.String("l1-api-key", "", "Level1 basic LLM API key")
	flags.String("l1-model", "", "Level1 basic LLM model")
	flags.String("l1-base-url", "", "Level1 basic LLM Base URL")

	flags.String("l2-api-key", "", "Level2 standard LLM API key")
	flags.String("l2-model", "", "Level2 standard LLM model")
	flags.String("l2-base-url", "", "Level2 standard LLM Base URL")

	flags.String("l3-api-key", "", "Level3 advanced LLM API key")
	flags.String("l3-model", "", "Level3 advanced LLM model")
	flags.String("l3-base-url", "", "Level3 advanced LLM Base URL")

	flags.String("tts-provider", "", "TTS provider")
	flags.String("tts-api-key", "", "TTS API key")
	flags.String("tts-model", "", "TTS model")
	flags.String("tts-base-url", "", "TTS Base URL")

	// flags.String("image-api-key", "", "Image LLM API key")
	// flags.String("image-model", "", "Image LLM model")
	// flags.String("image-base-url", "", "Image LLM Base URL")

	flags.MarkHidden("provider")

	// flags.MarkHidden("l1-model")
	// flags.MarkHidden("l2-model")
	// flags.MarkHidden("l3-model")

	flags.MarkHidden("l1-api-key")
	flags.MarkHidden("l2-api-key")
	flags.MarkHidden("l3-api-key")

	flags.MarkHidden("l1-base-url")
	flags.MarkHidden("l2-base-url")
	flags.MarkHidden("l3-base-url")

	// flags.MarkHidden("image-model")
	// flags.MarkHidden("image-api-key")
	// flags.MarkHidden("image-base-url")

	flags.String("image-viewer", "", "Image viewer")

	flags.MarkHidden("image-viewer")

	//
	flags.String("log", "", "Log all debugging information to a file")

	flags.Bool("quiet", false, "Operate quietly. Only show final response")
	flags.Bool("verbose", false, "Show progress and debugging information")
	flags.Bool("internal", false, "Enable internal agents and tools")

	flags.MarkHidden("internal")

	//
	flags.String("role", "system", "Specify a role for the prompt")
	flags.String("prompt", "", "Specify context instruction")

	flags.BoolVar(&internal.DryRun, "dry-run", false, "Enable dry run mode. No API call will be made")
	flags.StringVar(&internal.DryRunContent, "dry-run-content", "", "Content returned for dry run")

	flags.MarkHidden("log")

	flags.MarkHidden("role")
	flags.MarkHidden("prompt")

	flags.MarkHidden("dry-run")
	flags.MarkHidden("dry-run-content")

	//
	flags.BoolP("interactive", "i", false, "Interactive mode")
	flags.String("shell", os.Getenv("SHELL"), "Shell to use for interactive mode")

	flags.StringP("workspace", "w", "", "Workspace directory")

	// TODO
	flags.Bool("watch", false, "Watch the workspace directory and respond to embedded ai requests in files")
	flags.Bool("pb-watch", false, "Watch system clipboard and respond to embedded ai requests. Copy output to clipboard")

	flags.Bool("hub", false, "Start hub services")
	flags.String("hub-address", "localhost:58080", "Hub service host:port")

	flags.MarkHidden("workspace")
	flags.MarkHidden("watch")
	flags.MarkHidden("pb-watch")

	flags.Int("max-turns", 16, "Max number of turns")
	flags.Int("max-time", 3600, "Max number of seconds for timeout")

	flags.MarkHidden("max-time")

	// agent specific flags
	// db
	flags.String("sql-db-host", "", "Database host")
	flags.String("sql-db-port", "", "Database port")
	flags.String("sql-db-username", "", "Database username")
	flags.String("sql-db-password", "", "Database password")
	flags.String("sql-db-name", "", "Database name")

	flags.MarkHidden("sql-db-host")
	flags.MarkHidden("sql-db-port")
	flags.MarkHidden("sql-db-username")
	flags.MarkHidden("sql-db-password")
	flags.MarkHidden("sql-db-name")

	// doc
	flags.VarP(newTemplateValue("", &internal.TemplateFile), "template", "", "Document template file")

	flags.MarkHidden("template")

	// mcp - this is for mcp, but we need to define it here
	flags.Int("port", 0, "Port to run the server")
	flags.String("host", "localhost", "Host to bind the server")

	flags.MarkHidden("port")
	flags.MarkHidden("host")
}
