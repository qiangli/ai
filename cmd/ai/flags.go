package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go"
	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal"
)

// Output format type
type outputValue string

func newOutputValue(val string, p *string) *outputValue {
	*p = val
	return (*outputValue)(p)
}
func (s *outputValue) Set(val string) error {
	// TODO json
	for _, v := range []string{"text", "json", "markdown"} {
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

	flags.String("agent", "ask", "Specify the agent/command to use. Same as @agent/command")

	//
	flags.String("editor", "vi", "Specify editor to use")

	//
	flags.StringP("workspace", "w", "", "Workspace directory")

	// input
	flags.String("message", "", "Specify input message. Overrides all other input methods")

	flags.String("input", "", "Read input message from a file")
	flags.VarP(newFilesValue([]string{}, &internal.InputFiles), "file", "", `Read input from files.  May be given multiple times to add multiple file content`)
	flags.Bool("stdin", false, "Read input message from stdin. Alternatively, use '-'")
	flags.Bool("pb-read", false, "Read input from the clipboard. Alternatively, use '{'")
	flags.Bool("pb-read-wait", false, "Read input from the clipboard and wait for confirmation. Alternatively, use '{{'")

	// output
	flags.Bool("pb-write", false, "Copy output to the clipboard. Alternatively, use '}'")
	flags.Bool("pb-write-append", false, "Append output to the clipboard. Alternatively, use '}}'")
	flags.StringVarP(&internal.OutputFlag, "output", "o", "", "Save final response to a file.")

	flags.Var(newOutputValue("markdown", &internal.FormatFlag), "format", "Output format, one of text, json, or markdown.")

	// mcp
	flags.String("mcp-server-url", "http://localhost:58080/sse", "MCP server URL")

	// LLM
	flags.String("api-key", "", "LLM API key")
	flags.String("model", openai.ChatModelGPT4o, "LLM model")
	flags.String("base-url", "https://api.openai.com/v1/", "LLM Base URL")

	flags.String("l1-api-key", "", "Level1 basic LLM API key")
	flags.String("l1-model", openai.ChatModelGPT4oMini, "Level1 basic LLM model")
	flags.String("l1-base-url", "", "Level1 basic LLM Base URL")

	flags.String("l2-api-key", "", "Level2 standard LLM API key")
	flags.String("l2-model", openai.ChatModelGPT4o, "Level2 standard LLM model")
	flags.String("l2-base-url", "", "Level2 standard LLM Base URL")

	flags.String("l3-api-key", "", "Level3 advanced LLM API key")
	flags.String("l3-model", openai.ChatModelO1Mini, "Level3 advanced LLM model")
	flags.String("l3-base-url", "", "Level3 advanced LLM Base URL")

	flags.String("image-api-key", "", "Image LLM API key")
	flags.String("image-model", openai.ImageModelDallE3, "Image LLM model")
	flags.String("image-base-url", "", "Image LLM Base URL")
	flags.String("image-viewer", "", "Image viewer")

	flags.MarkHidden("l1-api-key")
	flags.MarkHidden("l2-api-key")
	flags.MarkHidden("l3-api-key")
	flags.MarkHidden("l1-base-url")
	flags.MarkHidden("l2-base-url")
	flags.MarkHidden("l3-base-url")

	flags.MarkHidden("image-api-key")
	flags.MarkHidden("image-base-url")
	flags.MarkHidden("image-viewer")

	//
	flags.String("log", "", "Log all debugging information to a file")
	flags.Bool("quiet", false, "Operate quietly. Only show final response")
	flags.Bool("verbose", false, "Show progress and debugging information")
	flags.Bool("internal", false, "Enable internal agents and tools")
	flags.Bool("unsafe", false, "Skip command security check to allow unsafe operations. Use with caution")

	//
	flags.String("role", "system", "Specify the role for the prompt")
	flags.String("role-prompt", "", "Specify the content for the prompt")

	flags.BoolVar(&internal.DryRun, "dry-run", false, "Enable dry run mode. No API call will be made")
	flags.StringVar(&internal.DryRunContent, "dry-run-content", "", "Content returned for dry run")

	// flags.Bool("no-meta-prompt", false, "Disable auto generation of system prompt")

	flags.BoolP("interactive", "i", false, "Interactive mode to run, edit, or copy generated code")

	flags.Bool("watch", false, "Watch the workspace directory and respond to embedded ai requests in files")

	flags.Int("max-turns", 32, "Max number of turns")
	flags.Int("max-time", 3600, "Max number of seconds for timeout")

	// agent specific flags
	// db
	flags.String("sql-db-host", "", "Database host")
	flags.String("sql-db-port", "", "Database port")
	flags.String("sql-db-username", "", "Database username")
	flags.String("sql-db-password", "", "Database password")
	flags.String("sql-db-name", "", "Database name")

	// doc
	flags.VarP(newTemplateValue("", &internal.TemplateFile), "template", "", "Document template file")

	// mcp - this is for mcp, but we need to define it here
	flags.Int("port", 0, "Port to run the server")
	flags.String("host", "localhost", "Host to bind the server")
	flags.MarkHidden("port")
	flags.MarkHidden("host")

	// hide flags
	flags.MarkHidden("editor")

	flags.MarkHidden("sql-db-host")
	flags.MarkHidden("sql-db-port")
	flags.MarkHidden("sql-db-username")
	flags.MarkHidden("sql-db-password")
	flags.MarkHidden("sql-db-name")

	flags.MarkHidden("role")
	flags.MarkHidden("role-prompt")

	flags.MarkHidden("dry-run")
	flags.MarkHidden("dry-run-content")

	flags.MarkHidden("log")

	flags.MarkHidden("interactive")
}
