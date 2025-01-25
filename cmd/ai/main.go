package main

import (
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
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/util"
)

func handle(cmd *cobra.Command, args []string) error {
	setLogLevel()

	fileLog, err := setLogOutput()
	if err != nil {
		return err
	}
	defer func() {
		if fileLog != nil {
			fileLog.Close()
		}
	}()

	cfg := getConfig(cmd, args)

	log.Debugf("Config: %+v %+v %+v\n", cfg, cfg.LLM, cfg.LLM.Sql.DBConfig)

	//
	command := cfg.Command

	// interactive mode
	// $ ai -i or $ ai --interactive
	// TODO: implement interactive mode
	if cfg.LLM.Interactive {
		// return shell.Bash(cfg.LLM)
		return fmt.Errorf("interactive mode not implemented yet")
	}

	// $ ai
	if command == "" && len(cfg.Args) == 0 {
		return cmd.Help()
	}

	// special commands
	if command != "" {
		// exact match with no message content
		// $ ai /
		// $ ai @
		// $ ai info
		// $ ai setup
		// $ ai help
		if len(cfg.Args) == 0 {
			switch command {
			case "/":
				return agent.ListCommands(cfg)
			case "list-commands":
				return agent.ListCommands(cfg)
			case "@":
				return agent.ListAgents(cfg)
			case "info":
				return agent.Info(cfg)
			case "setup":
				return agent.Setup(cfg)
			case "help":
				return Help(cmd)
			}
		}
	}

	if err := agent.HandleCommand(cfg); err != nil {
		log.Errorln(err)
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "ai [OPTIONS] AGENT [message...]",
	Short: "AI command line tool",
	Long: `AI Command Line Tool

	`,
	Example: resource.GetUserExample(),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handle(cmd, args)
	},
}

var cfgFile string
var formatFlag string
var outputFlag string

var inputFiles []string
var docTemplate string

func init() {
	defaultCfg := os.Getenv("AI_CONFIG")
	// default: ~/.ai/config.yaml
	homeDir := util.HomeDir()
	if defaultCfg == "" {
		if homeDir != "" {
			defaultCfg = filepath.Join(homeDir, ".ai", "config.yaml")
		}
	}

	//
	rootCmd.Flags().StringVar(&cfgFile, "config", defaultCfg, "config file")
	rootCmd.Flags().BoolVar(&internal.DryRun, "dry-run", false, "Enable dry run mode. No API call will be made")
	rootCmd.Flags().StringVar(&internal.DryRunContent, "dry-run-content", "", "Content returned for dry run")

	// Define flags with dashes
	rootCmd.Flags().StringP("workspace", "w", "", "Workspace directory")

	rootCmd.Flags().String("api-key", "", "LLM API key")
	rootCmd.Flags().String("model", openai.ChatModelGPT4o, "LLM model")
	rootCmd.Flags().String("base-url", "https://api.openai.com/v1/", "LLM Base URL")

	rootCmd.Flags().String("l1-api-key", "", "Level1 basic LLM API key")
	rootCmd.Flags().String("l1-model", openai.ChatModelGPT4oMini, "Level1 basic LLM model")
	rootCmd.Flags().String("l1-base-url", "", "Level1 basic LLM Base URL")

	rootCmd.Flags().String("l2-api-key", "", "Level2 standard LLM API key")
	rootCmd.Flags().String("l2-model", openai.ChatModelGPT4o, "Level2 standard LLM model")
	rootCmd.Flags().String("l2-base-url", "", "Level2 standard LLM Base URL")

	rootCmd.Flags().String("l3-api-key", "", "Level3 advanced LLM API key")
	rootCmd.Flags().String("l3-model", openai.ChatModelO1Mini, "Level3 advanced LLM model")
	rootCmd.Flags().String("l3-base-url", "", "Level3 advanced LLM Base URL")

	rootCmd.Flags().MarkHidden("l1-api-key")
	rootCmd.Flags().MarkHidden("l2-api-key")
	rootCmd.Flags().MarkHidden("l3-api-key")
	rootCmd.Flags().MarkHidden("l1-base-url")
	rootCmd.Flags().MarkHidden("l2-base-url")
	rootCmd.Flags().MarkHidden("l3-base-url")

	rootCmd.Flags().Bool("verbose", false, "Show debugging information")
	rootCmd.Flags().Bool("quiet", false, "Operate quietly")
	rootCmd.Flags().String("editor", "vi", "Specify editor to use")

	rootCmd.Flags().String("role", "system", "Specify the role for the prompt")
	rootCmd.Flags().String("role-prompt", "", "Specify the content for the prompt")

	rootCmd.Flags().BoolP("no-meta-prompt", "n", false, "Disable auto generation of system prompt")

	rootCmd.Flags().BoolP("interactive", "i", false, "Interactive mode to run, edit, or copy generated code")

	rootCmd.Flags().Bool("pb-read", false, "Read input from the clipboard. Alternatively, append '=' to the command")
	rootCmd.Flags().Bool("pb-write", false, "Copy output to the clipboard. Alternatively, append '=+' to the command")

	rootCmd.Flags().VarP(newFilesValue([]string{}, &inputFiles), "file", "", `Read input from files.  May be given multiple times to add multiple file content`)

	rootCmd.Flags().String("log", "", "Log all debugging information to a file")
	rootCmd.Flags().Bool("trace", false, "Trace API calls")

	//
	rootCmd.Flags().MarkHidden("role")
	rootCmd.Flags().MarkHidden("role-prompt")
	rootCmd.Flags().MarkHidden("dry-run")
	rootCmd.Flags().MarkHidden("dry-run-content")
	rootCmd.Flags().MarkHidden("trace")

	rootCmd.Flags().Var(newOutputValue("markdown", &formatFlag), "format", "Output format, must be either raw or markdown.")
	rootCmd.Flags().StringVar(&outputFlag, "output", "", "Save final response to a file.")

	// agent specific flags
	// db
	rootCmd.Flags().String("sql-db-host", "", "Database host")
	rootCmd.Flags().String("sql-db-port", "", "Database port")
	rootCmd.Flags().String("sql-db-username", "", "Database username")
	rootCmd.Flags().String("sql-db-password", "", "Database password")
	rootCmd.Flags().String("sql-db-name", "", "Database name")

	// doc
	rootCmd.Flags().VarP(newTemplateValue("", &docTemplate), "doc-template", "", "Document template file")

	// Bind the flags to viper using underscores
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	// Bind the flags to viper using dots
	viper.BindPFlag("sql.db-name", rootCmd.Flags().Lookup("sql-db-name"))
	viper.BindPFlag("sql.db-host", rootCmd.Flags().Lookup("sql-db-host"))
	viper.BindPFlag("sql.db-port", rootCmd.Flags().Lookup("sql-db-port"))
	viper.BindPFlag("sql.db-username", rootCmd.Flags().Lookup("sql-db-username"))
	viper.BindPFlag("sql.db-password", rootCmd.Flags().Lookup("sql-db-password"))

	viper.BindPFlag("git.short", rootCmd.Flags().Lookup("git-short"))

	viper.AutomaticEnv()
	viper.SetEnvPrefix("ai")
	viper.BindEnv("api-key", "AI_API_KEY", "OPENAI_API_KEY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	//
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SetUsageTemplate(rootUsageTemplate)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			internal.Exit(err)
		}
		viper.AddConfigPath(home)
		viper.SetConfigName(".ai")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Error reading config file: %s\n", err)
	}
}

func getConfig(cmd *cobra.Command, args []string) *internal.AppConfig {
	var cfg internal.LLMConfig
	var app = internal.AppConfig{
		LLM:    &cfg,
		Role:   viper.GetString("role"),
		Prompt: viper.GetString("role_prompt"),
	}

	app.Me = "ME"
	app.Files = inputFiles
	app.Format = formatFlag
	app.Output = outputFlag
	app.ConfigFile = viper.ConfigFileUsed()
	app.CommandPath = cmd.CommandPath()

	//
	app.Template = docTemplate

	cfg.Workspace = viper.GetString("workspace")

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

	cfg.Debug = viper.GetBool("verbose")

	app.Editor = viper.GetString("editor")

	cfg.Interactive = viper.GetBool("interactive")
	noMeta := viper.GetBool("no_meta_prompt")
	cfg.MetaPrompt = !noMeta

	//
	cfg.WorkDir, _ = os.Getwd()

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
		// check for valid command
		valid := func() bool {
			misc := []string{"info", "setup", "help", "list-commands"}
			if strings.HasPrefix(newArgs[0], "/") {
				return true
			}
			if strings.HasPrefix(newArgs[0], "@") {
				return true
			}
			for _, v := range misc {
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
	cfg.Sql = &internal.SQLConfig{
		DBConfig: dbCfg,
	}

	//
	gitConfig := &internal.GitConfig{}
	cfg.Git = gitConfig

	return &app
}

func setLogLevel() {
	quiet := viper.GetBool("quiet")
	if quiet {
		log.SetLogLevel(log.Quiet)
		return
	}
	debug := viper.GetBool("verbose")
	if debug {
		log.SetLogLevel(log.Verbose)
	}

	// trace
	log.Trace = viper.GetBool("trace")

}

func setLogOutput() (*log.FileWriter, error) {
	pathname := viper.GetString("log")
	if pathname != "" {
		f, err := log.NewFileWriter(pathname)
		if err != nil {
			return nil, err
		}
		log.SetLogOutput(f)
		return f, nil
	}
	return nil, nil
}

func main() {
	cobra.OnInitialize(initConfig)

	if err := rootCmd.Execute(); err != nil {
		internal.Exit(err)
	}
}
