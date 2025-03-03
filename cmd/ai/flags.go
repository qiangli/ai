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
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/internal/util"
)

var builtinCommands = []string{"info", "setup", "help", "list-commands", "list-agents", "list-tools", "commands", "agents", "tools"}

var cfgFile string
var formatFlag string
var outputFlag string

var inputFiles []string
var docTemplate string

// Output format type
type outputValue string

func newOutputValue(val string, p *string) *outputValue {
	*p = val
	return (*outputValue)(p)
}
func (s *outputValue) Set(val string) error {
	for _, v := range []string{"raw", "markdown"} {
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

func addFlags(cmd *cobra.Command) {
	defaultCfg := os.Getenv("AI_CONFIG")
	// default: ~/.ai/config.yaml
	homeDir := util.HomeDir()
	if defaultCfg == "" {
		if homeDir != "" {
			defaultCfg = filepath.Join(homeDir, ".ai", "config.yaml")
		}
	}

	flags := cmd.Flags()
	//
	flags.StringVar(&cfgFile, "config", defaultCfg, "config file")

	//
	flags.StringP("workspace", "w", "", "Workspace directory")

	flags.String("message", "", "Specify input message. Overrides all other input methods")
	flags.String("editor", "vi", "Specify editor to use")
	flags.VarP(newFilesValue([]string{}, &inputFiles), "file", "", `Read input from files.  May be given multiple times to add multiple file content`)

	flags.Bool("pb-read", false, "Read input from the clipboard. Alternatively, append '=' to the command")
	flags.Bool("pb-write", false, "Copy output to the clipboard. Alternatively, append '=+' to the command")

	flags.Var(newOutputValue("markdown", &formatFlag), "format", "Output format, must be either raw or markdown.")
	flags.StringVarP(&outputFlag, "output", "o", "", "Save final response to a file.")

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
	flags.Bool("verbose", false, "Show debugging information")
	flags.Bool("quiet", false, "Operate quietly")
	flags.String("log", "", "Log all debugging information to a file")
	flags.Bool("trace", false, "Trace API calls")

	flags.String("role", "system", "Specify the role for the prompt")
	flags.String("role-prompt", "", "Specify the content for the prompt")

	flags.BoolVar(&internal.DryRun, "dry-run", false, "Enable dry run mode. No API call will be made")
	flags.StringVar(&internal.DryRunContent, "dry-run-content", "", "Content returned for dry run")

	flags.Bool("no-meta-prompt", false, "Disable auto generation of system prompt")

	flags.BoolP("interactive", "i", false, "Interactive mode to run, edit, or copy generated code")

	flags.Int("max-turns", 32, "Max number of turns")
	flags.Int("max-time", 3600, "Max number of seconds for timeout")

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
	flags.VarP(newTemplateValue("", &docTemplate), "doc-template", "", "Document template file")

	//
	flags.MarkHidden("message")
	flags.MarkHidden("role")
	flags.MarkHidden("role-prompt")
	flags.MarkHidden("dry-run")
	flags.MarkHidden("dry-run-content")
	flags.MarkHidden("trace")
	flags.MarkHidden("log")
	flags.MarkHidden("interactive")
	flags.MarkHidden("no-meta-prompt")

	// Bind the flags to viper using underscores
	flags.VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	// Bind the flags to viper using dots
	viper.BindPFlag("sql.db-name", flags.Lookup("sql-db-name"))
	viper.BindPFlag("sql.db-host", flags.Lookup("sql-db-host"))
	viper.BindPFlag("sql.db-port", flags.Lookup("sql-db-port"))
	viper.BindPFlag("sql.db-username", flags.Lookup("sql-db-username"))
	viper.BindPFlag("sql.db-password", flags.Lookup("sql-db-password"))

	viper.BindPFlag("git.short", flags.Lookup("git-short"))

	viper.AutomaticEnv()
	viper.SetEnvPrefix("ai")
	viper.BindEnv("api-key", "AI_API_KEY", "OPENAI_API_KEY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	//
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetUsageTemplate(rootUsageTemplate)
}

func configure(cmd *cobra.Command, args []string) *internal.AppConfig {
	var cfg internal.LLMConfig
	var app = internal.AppConfig{
		LLM:    &cfg,
		Role:   viper.GetString("role"),
		Prompt: viper.GetString("role_prompt"),
	}

	app.Me = "👤 ME"
	app.Files = inputFiles
	app.Format = formatFlag
	app.Output = outputFlag
	app.ConfigFile = viper.ConfigFileUsed()
	app.CommandPath = cmd.CommandPath()

	app.Message = viper.GetString("message")

	//
	app.Template = docTemplate

	app.Workspace = viper.GetString("workspace")

	//
	cfg.ApiKey = viper.GetString("api_key")
	cfg.Model = viper.GetString("model")
	cfg.BaseUrl = viper.GetString("base_url")

	cfg.L1Model = viper.GetString("l1_model")
	cfg.L1BaseUrl = viper.GetString("l1_base_url")
	cfg.L1ApiKey = viper.GetString("l1_api_key")
	cfg.L2Model = viper.GetString("l2_model")
	cfg.L2BaseUrl = viper.GetString("l2_base_url")
	cfg.L2ApiKey = viper.GetString("l2_api_key")
	cfg.L3Model = viper.GetString("l3_model")
	cfg.L3BaseUrl = viper.GetString("l3_base_url")
	cfg.L3ApiKey = viper.GetString("l3_api_key")
	if cfg.L1Model == "" {
		cfg.L1Model = cfg.Model
	}
	if cfg.L2Model == "" {
		cfg.L2Model = cfg.Model
	}
	if cfg.L3Model == "" {
		cfg.L3Model = cfg.Model
	}
	if cfg.L1ApiKey == "" {
		cfg.L1ApiKey = cfg.ApiKey
	}
	if cfg.L2ApiKey == "" {
		cfg.L2ApiKey = cfg.ApiKey
	}
	if cfg.L3ApiKey == "" {
		cfg.L3ApiKey = cfg.ApiKey
	}
	if cfg.L1BaseUrl == "" {
		cfg.L1BaseUrl = cfg.BaseUrl
	}
	if cfg.L2BaseUrl == "" {
		cfg.L2BaseUrl = cfg.BaseUrl
	}
	if cfg.L3BaseUrl == "" {
		cfg.L3BaseUrl = cfg.BaseUrl
	}

	cfg.ImageModel = viper.GetString("image_model")
	cfg.ImageBaseUrl = viper.GetString("image_base_url")
	cfg.ImageApiKey = viper.GetString("image_api_key")

	//
	app.Debug = viper.GetBool("verbose")
	app.Editor = viper.GetString("editor")
	app.Interactive = viper.GetBool("interactive")
	noMeta := viper.GetBool("no_meta_prompt")
	app.MetaPrompt = !noMeta
	app.WorkDir, _ = os.Getwd()

	app.MaxTurns = viper.GetInt("max_turns")
	app.MaxTime = viper.GetInt("max_time")

	// special char sequence handling
	var pbRead = viper.GetBool("pb_read")
	var pbWrite = viper.GetBool("pb_write")
	var isStdin, isClipin, isClipout bool
	newArgs := args

	// parse special char sequence
	if len(args) > 0 {
		for i := len(args) - 1; i >= 0; i-- {
			lastArg := args[i]

			if lastArg == agent.StdinInputRedirect {
				isStdin = true
			} else if lastArg == agent.ClipboardInputRedirect {
				isClipin = true
			} else if lastArg == agent.ClipboardOutputRedirect {
				isClipout = true
			} else {
				// check for suffix for cases where the special char is not the last arg
				// but is part of the last arg
				if strings.HasSuffix(lastArg, agent.StdinInputRedirect) {
					isStdin = true
					args[i] = strings.TrimSuffix(lastArg, agent.StdinInputRedirect)
				} else if strings.HasSuffix(lastArg, agent.ClipboardInputRedirect) {
					isClipin = true
					args[i] = strings.TrimSuffix(lastArg, agent.ClipboardInputRedirect)
				} else if strings.HasSuffix(lastArg, agent.ClipboardOutputRedirect) {
					isClipout = true
					args[i] = strings.TrimSuffix(lastArg, agent.ClipboardOutputRedirect)
				}
				newArgs = args[:i+1]
				break
			}
		}
	}

	app.Stdin = isStdin
	app.Clipin = isClipin || pbRead
	app.Clipout = isClipout || pbWrite

	// command and args
	if len(newArgs) > 0 {
		// check for valid sub command
		valid := func() bool {
			if strings.HasPrefix(newArgs[0], "/") {
				return true
			}
			if strings.HasPrefix(newArgs[0], "@") {
				return true
			}
			for _, v := range builtinCommands {
				if v == newArgs[0] && len(newArgs) == 1 {
					return true
				}
			}
			return false
		}
		if valid() {
			app.Command = newArgs[0]
			if len(newArgs) > 1 {
				app.Args = newArgs[1:]
			}
		} else {
			app.Command = ""
			app.Args = newArgs
		}

	}

	// sql db
	dbCfg := &internal.DBConfig{}
	dbCfg.Host = viper.GetString("sql.db_host")
	dbCfg.Port = viper.GetString("sql.db_port")
	dbCfg.Username = viper.GetString("sql.db_username")
	dbCfg.Password = viper.GetString("sql.db_password")
	dbCfg.DBName = viper.GetString("sql.db_name")
	app.Db = dbCfg
	//
	gitConfig := &internal.GitConfig{}
	app.Git = gitConfig

	return &app
}
