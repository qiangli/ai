package hub

import (
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	fangs "github.com/spf13/viper"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

func parseFlags(line string, cfg *api.AppConfig) error {
	args, err := shellwords.Parse(line)
	if err != nil {
		return err
	}

	viper := fangs.New()
	if cfg.ConfigFile != "" {
		viper.SetConfigFile(cfg.ConfigFile)
		viper.AutomaticEnv()
		if err := viper.ReadInConfig(); err != nil {
			log.Debugf("Error reading config file: %s\n", err)
			return err
		}
	}

	var run = func(cmd *cobra.Command, args []string) error {
		internal.ParseConfig(viper, cfg, args)
		return nil
	}
	var agentCmd = &cobra.Command{
		Args: cobra.ArbitraryArgs,
		RunE: run,
	}

	addAgentFlags(agentCmd, cfg)
	agentCmd.SetArgs(args)

	// Bind the flags to viper using underscores
	agentCmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	return agentCmd.Execute()
}

// add a subset of supported AI flags for chatbots
// agent
// new thread
// models
func addAgentFlags(cmd *cobra.Command, cfg *api.AppConfig) {
	flags := cmd.Flags()

	// --agent agent/command or @agent/command
	flags.StringP("agent", "a", cfg.Agent, "Specify the agent to use or @agent")

	flags.String("format", cfg.Format, "Output format: raw, text, json, markdown, or tts.")

	// security
	flags.Bool("unsafe", cfg.Unsafe, "Skip command security check to allow unsafe operations. Use with caution")

	// conversation
	flags.BoolP("new", "n", cfg.New, "Start a new conversation")
	flags.Int("max-history", cfg.MaxHistory, "Max number of historic messages")
	flags.Int("max-span", cfg.MaxSpan, "How far in minutes to go back in time for historic messages")

	// LLM
	flags.StringP("models", "m", cfg.Models, "LLM model alias defined in the models directory")

	// if cfg.LLM == nil {
	// 	cfg.LLM = &api.LLMConfig{}
	// }
	// flags.String("provider", cfg.LLM.Provider, "LLM provider")
	// flags.String("api-key", cfg.LLM.ApiKey, "LLM API key")
	// flags.String("model", cfg.LLM.Model, "LLM default model")
	// flags.String("base-url", cfg.LLM.BaseUrl, "LLM Base URL")

	// assume cfg.TTS not nil
	// if cfg.TTS == nil {
	// 	cfg.TTS = &api.TTSConfig{}
	// }
	// flags.String("tts-provider", cfg.TTS.Provider, "TTS provider")
	// flags.String("tts-api-key", cfg.TTS.ApiKey, "TTS API key")
	// flags.String("tts-model", cfg.TTS.Model, "TTS model")
	// flags.String("tts-base-url", cfg.TTS.BaseUrl, "TTS Base URL")

	//
	flags.Int("max-turns", cfg.MaxTurns, "Max number of turns")
	flags.Int("max-time", cfg.MaxTime, "Max number of seconds for timeout")
}
