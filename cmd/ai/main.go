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
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/util"
)

type AppConfig struct {
	LLM *internal.Config

	Role    string
	Message string
}

var rootCmd = &cobra.Command{
	Use:   "ai [OPTIONS] COMMAND [message...]",
	Short: "AI command line tool",
	Long: `AI Command Line Tool

	`,
	Example: resource.GetUserExample(),
	RunE: func(cmd *cobra.Command, args []string) error {
		setLogLevel()

		cfg := getConfig(args)

		log.Debugf("LLM config: %+v\n", cfg.LLM)

		command := cfg.LLM.Command
		switch command {
		case "list":
			return internal.ListCommand(cfg.LLM)
		case "info":
			return internal.InfoCommand(cfg.LLM)
		case "help":
			return Help(cmd)
		}

		// remote - LLM API call
		if strings.HasPrefix(command, "/") {
			return internal.SlashCommand(cfg.LLM, cfg.Role, cfg.Message)
		}
		if strings.HasPrefix(command, "@") {
			return internal.AgentCommand(cfg.LLM, cfg.Role, cfg.Message)
		}

		return cmd.Help()
	},
}

var cfgFile string

func init() {
	defaultCfg := os.Getenv("AI_CONFIG")
	// default: ~/.ai/config.yaml
	if defaultCfg == "" {
		homeDir := util.HomeDir()
		if homeDir != "" {
			defaultCfg = filepath.Join(homeDir, ".ai", "config.yaml")
		}
	}
	rootCmd.Flags().StringVar(&cfgFile, "config", defaultCfg, "config file")

	// Define flags with dashes
	rootCmd.Flags().String("api-key", "", "LLM API key")
	rootCmd.Flags().String("model", openai.ChatModelGPT4o, "LLM model")
	rootCmd.Flags().String("base-url", "https://api.openai.com/v1/", "LLM Base URL")
	rootCmd.Flags().Bool("verbose", false, "Show debugging information")
	rootCmd.Flags().Bool("quiet", false, "Operate quietly")
	rootCmd.Flags().Bool("dry-run", false, "Enable dry run mode. No API call will be made")
	rootCmd.Flags().String("dry-run-content", "", "Content returned for dry run")
	rootCmd.Flags().String("editor", "vi", "Specify editor to use")

	rootCmd.Flags().String("role", "system", "Prompt role")
	rootCmd.Flags().String("role-content", "", "Prompt role content (default auto)")

	rootCmd.Flags().BoolP("interactive", "i", false, "Interactive mode to run, edit, or copy generated code")

	rootCmd.Flags().Bool("pb-read", false, "Read input from the clipboard. Alternatively, append '=' to the command")
	rootCmd.Flags().Bool("pb-write", false, "Copy output to the clipboard. Alternatively, append '=+' to the command")

	// Bind the flags to viper using underscores
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	viper.AutomaticEnv()
	viper.SetEnvPrefix("ai")
	viper.BindEnv("api-key", "AI_API_KEY", "OPENAI_API_KEY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

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

func getConfig(args []string) *AppConfig {
	var cfg internal.Config

	cfg.ApiKey = viper.GetString("api_key")
	cfg.Model = viper.GetString("model")
	cfg.BaseUrl = viper.GetString("base_url")
	cfg.Debug = viper.GetBool("verbose")
	cfg.DryRun = viper.GetBool("dry_run")
	cfg.DryRunContent = viper.GetString("dry_run_content")
	cfg.Editor = viper.GetString("editor")

	cfg.Interactive = viper.GetBool("interactive")

	//
	cfg.WorkDir, _ = os.Getwd()

	// special char sequence handling
	var pbRead = viper.GetBool("pb_read")
	var pbWrite = viper.GetBool("pb_write")
	var isStdin, isClipin, isClipout bool
	newArgs := args

	if len(args) > 0 {
		for i := len(args) - 1; i >= 0; i-- {
			lastArg := args[i]

			if lastArg == internal.StdinInputRedirect {
				isStdin = true
			} else if lastArg == internal.ClipboardInputRedirect {
				isClipin = true
			} else if lastArg == internal.ClipboardOutputRedirect {
				isClipout = true
			} else {
				// check for suffix for cases where the special char is not the last arg
				// but is part of the last arg
				if strings.HasSuffix(lastArg, internal.StdinInputRedirect) {
					isStdin = true
					args[i] = strings.TrimSuffix(lastArg, internal.StdinInputRedirect)
				} else if strings.HasSuffix(lastArg, internal.ClipboardInputRedirect) {
					isClipin = true
					args[i] = strings.TrimSuffix(lastArg, internal.ClipboardInputRedirect)
				} else if strings.HasSuffix(lastArg, internal.ClipboardOutputRedirect) {
					isClipout = true
					args[i] = strings.TrimSuffix(lastArg, internal.ClipboardOutputRedirect)
				}
				newArgs = args[:i+1]
				break
			}
		}
	}

	cfg.Stdin = isStdin
	cfg.Clipin = isClipin || pbRead
	cfg.Clipout = isClipout || pbWrite

	if len(newArgs) > 0 {
		cfg.Command = newArgs[0]
		if len(newArgs) > 1 {
			cfg.Args = newArgs[1:]
		}
	}

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
}

func main() {
	cobra.OnInitialize(initConfig)

	if err := rootCmd.Execute(); err != nil {
		internal.Exit(err)
	}
}
