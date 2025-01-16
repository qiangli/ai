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
	"github.com/qiangli/ai/internal/db"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/shell"
	"github.com/qiangli/ai/internal/util"
)

type AppConfig struct {
	LLM *llm.Config

	Role    string
	Message string
}

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

	log.Debugf("Config: %+v %+v %+v\n", cfg, cfg.LLM, cfg.LLM.DBConfig)

	// set global flags
	internal.Debug = cfg.LLM.Debug
	internal.DryRun = cfg.LLM.DryRun
	internal.DryRunContent = cfg.LLM.DryRunContent
	internal.WorkDir = cfg.LLM.WorkDir

	//
	command := cfg.LLM.Command

	// interactive mode
	// $ ai -i or $ ai --interactive
	if cfg.LLM.Interactive {
		return shell.Bash(cfg.LLM)
	}

	// $ ai
	if command == "" && len(cfg.LLM.Args) == 0 {
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
		if len(cfg.LLM.Args) == 0 {
			switch command {
			case "/":
				return agent.ListCommands(cfg.LLM)
			case "@":
				return agent.ListAgents(cfg.LLM)
			case "info":
				return agent.Info(cfg.LLM)
			case "setup":
				return agent.Setup(cfg.LLM)
			case "help":
				return Help(cmd)
			}
		}
	}

	if err := agent.HandleCommand(cfg.LLM, cfg.Role, cfg.Message); err != nil {
		log.Errorln(err)
	}
	return nil
}

var rootCmd = &cobra.Command{
	Use:   "ai [OPTIONS] COMMAND [message...]",
	Short: "AI command line tool",
	Long: `AI Command Line Tool

	`,
	Example: resource.GetUserExample(),
	RunE: func(cmd *cobra.Command, args []string) error {
		return handle(cmd, args)
	},
}

var cfgFile string

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
	rootCmd.Flags().Bool("dry-run", false, "Enable dry run mode. No API call will be made")
	rootCmd.Flags().String("dry-run-content", "", "Content returned for dry run")
	rootCmd.Flags().String("editor", "vi", "Specify editor to use")

	rootCmd.Flags().String("role", "system", "Specify the role for the prompt")
	rootCmd.Flags().String("role-content", "", "Specify the content for the prompt")

	rootCmd.Flags().BoolP("no-meta-prompt", "n", false, "Disable auto generation of system prompt")

	rootCmd.Flags().BoolP("interactive", "i", false, "Interactive mode to run, edit, or copy generated code")

	rootCmd.Flags().Bool("pb-read", false, "Read input from the clipboard. Alternatively, append '=' to the command")
	rootCmd.Flags().Bool("pb-write", false, "Copy output to the clipboard. Alternatively, append '=+' to the command")

	rootCmd.Flags().String("log", "", "Log all debugging information to a file")
	rootCmd.Flags().Bool("trace", false, "Trace API calls")

	// db
	rootCmd.Flags().String("db-host", "", "Database host")
	rootCmd.Flags().String("db-port", "", "Database port")
	rootCmd.Flags().String("db-username", "", "Database username")
	rootCmd.Flags().String("db-password", "", "Database password")
	rootCmd.Flags().String("db-name", "", "Database name")

	//
	rootCmd.Flags().MarkHidden("role")
	rootCmd.Flags().MarkHidden("role-content")
	rootCmd.Flags().MarkHidden("dry-run")
	rootCmd.Flags().MarkHidden("dry-run-content")
	rootCmd.Flags().MarkHidden("trace")

	// TODO agent specific flags
	rootCmd.Flags().Bool("git-short", false, "Generate short one liner commit message")

	// Bind the flags to viper using underscores
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	viper.BindPFlag("db.name", rootCmd.Flags().Lookup("db-name"))
	viper.BindPFlag("db.host", rootCmd.Flags().Lookup("db-host"))
	viper.BindPFlag("db.port", rootCmd.Flags().Lookup("db-port"))
	viper.BindPFlag("db.username", rootCmd.Flags().Lookup("db-username"))
	viper.BindPFlag("db.password", rootCmd.Flags().Lookup("db-password"))

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

func getConfig(cmd *cobra.Command, args []string) *AppConfig {
	var cfg llm.Config

	cfg.Me = "ME"
	cfg.ConfigFile = viper.ConfigFileUsed()
	cfg.CommandPath = cmd.CommandPath()
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

	cfg.DryRun = viper.GetBool("dry_run")
	cfg.DryRunContent = viper.GetString("dry_run_content")

	cfg.Editor = viper.GetString("editor")

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

	cfg.Stdin = isStdin
	cfg.Clipin = isClipin || pbRead
	cfg.Clipout = isClipout || pbWrite

	// command and args
	if len(newArgs) > 0 {
		// check for valid command
		valid := func() bool {
			misc := []string{"info", "setup", "help"}
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
			cfg.Command = newArgs[0]
			if len(newArgs) > 1 {
				cfg.Args = newArgs[1:]
			}
		} else {
			cfg.Command = ""
			cfg.Args = newArgs
		}

	}

	// db
	dbCfg := &db.DBConfig{}

	dbCfg.Host = viper.GetString("db.host")
	dbCfg.Port = viper.GetString("db.port")
	dbCfg.Username = viper.GetString("db.username")
	dbCfg.Password = viper.GetString("db.password")
	dbCfg.DBName = viper.GetString("db.name")

	cfg.DBConfig = dbCfg

	//
	gitConfig := &llm.GitConfig{}
	cfg.Git = gitConfig
	gitConfig.Short = viper.GetBool("git_short")

	return &AppConfig{
		LLM:     &cfg,
		Role:    viper.GetString("role"),
		Message: viper.GetString("role_content"),
	}
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
