package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/cli/internal"
	"github.com/qiangli/ai/cli/internal/log"
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
	Example: internal.GetUserExample(),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := getConfig()
		setLogLevel(cfg.LLM.Debug)

		log.Debugf("LLM config: %+v\n", cfg.LLM)

		if len(args) > 0 {
			// local - no LLM API call
			if args[0] == "list" {
				return internal.ListCommand(cfg.LLM, args)
			}
			if args[0] == "info" {
				return internal.InfoCommand(cfg.LLM, args)
			}
			if args[0] == "help" {
				return internal.HelpCommand(cfg.LLM, args)
			}

			// remote - LLM API call
			if strings.HasPrefix(args[0], "/") {
				return internal.SlashCommand(cfg.LLM, args, cfg.Role, cfg.Message)
			}
			if strings.HasPrefix(args[0], "@") {
				return internal.AgentCommand(cfg.LLM, args, cfg.Role, cfg.Message)
			}
		}

		return cmd.Help()
	},
}

var cfgFile string

func init() {
	defaultCfg := os.Getenv("AI_CONFIG")
	// default: ~/.ai/config.yaml
	if defaultCfg == "" {
		homeDir := internal.HomeDir()
		if homeDir != "" {
			defaultCfg = filepath.Join(homeDir, ".ai", "config.yaml")
		}
	}
	rootCmd.Flags().StringVar(&cfgFile, "config", defaultCfg, "config file")

	// Define flags with dashes
	rootCmd.Flags().String("api-key", "", "LLM API key")
	rootCmd.Flags().String("model", openai.ChatModelGPT4o, "LLM model")
	rootCmd.Flags().String("base-url", "https://api.openai.com/v1/", "LLM Base URL")
	rootCmd.Flags().Bool("debug", false, "Enable verbose console output")
	rootCmd.Flags().Bool("dry-run", false, "Enable dry run mode. No API call will be made")
	rootCmd.Flags().String("dry-run-content", "", "Content will be returned for dry run")
	rootCmd.Flags().String("editor", "vi", "Specify the editor to use")

	rootCmd.Flags().String("role", "system", "Prompt role")
	rootCmd.Flags().String("role-content", "", "Prompt role content (default auto)")

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

	if err := viper.ReadInConfig(); err == nil {
		// FIXME
		log.Debugln("Using config file:", viper.ConfigFileUsed())
	}
}

func getConfig() *AppConfig {
	var cfg internal.Config

	cfg.ApiKey = viper.GetString("api_key")
	cfg.Model = viper.GetString("model")
	cfg.BaseUrl = viper.GetString("base_url")
	cfg.Debug = viper.GetBool("debug")
	cfg.DryRun = viper.GetBool("dry_run")
	cfg.DryRunContent = viper.GetString("dry_run_content")
	cfg.Editor = viper.GetString("editor")

	//
	cfg.WorkDir, _ = os.Getwd()

	return &AppConfig{
		LLM:     &cfg,
		Role:    viper.GetString("role"),
		Message: viper.GetString("role_content"),
	}
}

func setLogLevel(debug bool) {
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
