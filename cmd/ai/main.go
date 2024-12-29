package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/openai/openai-go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/cli/internal"
	"github.com/qiangli/ai/cli/internal/log"
)

var config internal.Config

var rootCmd = &cobra.Command{
	Use:   "ai [OPTIONS] COMMAND [message...]",
	Short: "AI command line tool",
	Long: `AI Command Line Tool

	`,
	Example: internal.GetUserExample(),
	RunE: func(cmd *cobra.Command, args []string) error {
		setConfig(&config)
		setLogLevel(config.Debug)

		if len(args) > 0 {
			// local - no LLM API call
			if args[0] == "list" {
				return internal.ListCommand(&config, args)
			}
			if args[0] == "info" {
				return internal.InfoCommand(&config, args)
			}
			if args[0] == "help" {
				return internal.HelpCommand(&config, args)
			}

			// remote - LLM API call
			if strings.HasPrefix(args[0], "/") {
				return internal.SlashCommand(&config, args)
			}
			if strings.HasPrefix(args[0], "@") {
				return internal.AgentCommand(&config, args)
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", defaultCfg, "config file")

	rootCmd.Flags().String("api-key", "", "LLM API key")
	rootCmd.Flags().String("model", openai.ChatModelGPT4o, "LLM model")
	rootCmd.Flags().String("base-url", "https://api.openai.com/v1/", "LLM Base URL")
	rootCmd.Flags().Bool("debug", false, "Enable verbose console output")
	rootCmd.Flags().Bool("dry-run", false, "Enable dry run mode. No API call will be made")
	rootCmd.Flags().String("dry-run-file", "", "Content of the file will be returned for dry run")
	rootCmd.Flags().String("editor", "vi", "Specify the editor to use")

	viper.BindPFlags(rootCmd.Flags())

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
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

func setConfig(cfg *internal.Config) {
	cfg.ApiKey = viper.GetString("api-key")
	cfg.Model = viper.GetString("model")
	cfg.BaseUrl = viper.GetString("base-url")
	cfg.Debug = viper.GetBool("debug")
	cfg.DryRun = viper.GetBool("dry-run")
	cfg.DryRunFile = viper.GetString("dry-run-file")
	cfg.Editor = viper.GetString("editor")
	//
	cfg.WorkDir, _ = os.Getwd()
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
