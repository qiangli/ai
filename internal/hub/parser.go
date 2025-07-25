package hub

import (
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/swarm/api"
)

func parseFlags(line string, cfg *api.AppConfig) error {
	args, err := shellwords.Parse(line)
	if err != nil {
		return err
	}

	var run = func(cmd *cobra.Command, args []string) error {
		parseAgentFlags(cmd, cfg)
		internal.ParseArgs(cfg, args, cfg.Agent)
		return nil
	}
	var agentCmd = &cobra.Command{
		Args: cobra.ArbitraryArgs,
		RunE: run,
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	addAgentFlags(agentCmd, cfg)
	agentCmd.SetArgs(args)

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

	// security
	flags.Bool("unsafe", cfg.Unsafe, "Skip command security check to allow unsafe operations. Use with caution")

	// conversation
	flags.BoolP("new", "n", cfg.New, "Start a new conversation")
	flags.Int("max-history", cfg.MaxHistory, "Max number of historic messages")
	flags.Int("max-span", cfg.MaxSpan, "How far in minutes to go back in time for historic messages")

	// LLM
	flags.StringP("models", "m", cfg.Models, "LLM model alias defined in the models directory")

	if cfg.LLM == nil {
		cfg.LLM = &api.LLMConfig{}
	}
	flags.String("provider", cfg.LLM.Provider, "LLM provider")
	flags.String("api-key", cfg.LLM.ApiKey, "LLM API key")
	flags.String("model", cfg.LLM.Model, "LLM default model")
	flags.String("base-url", cfg.LLM.BaseUrl, "LLM Base URL")

	// assume cfg.TTS not nil
	if cfg.TTS == nil {
		cfg.TTS = &api.TTSConfig{}
	}
	flags.String("tts-provider", cfg.TTS.Provider, "TTS provider")
	flags.String("tts-api-key", cfg.TTS.ApiKey, "TTS API key")
	flags.String("tts-model", cfg.TTS.Model, "TTS model")
	flags.String("tts-base-url", cfg.TTS.BaseUrl, "TTS Base URL")

	//
	flags.Int("max-turns", cfg.MaxTurns, "Max number of turns")
	flags.Int("max-time", cfg.MaxTime, "Max number of seconds for timeout")
}
func parseAgentFlags(cmd *cobra.Command, cfg *api.AppConfig) {
	flags := cmd.Flags()

	// --agent agent/command or @agent/command
	cfg.Agent, _ = flags.GetString("agent")

	// security
	cfg.Unsafe, _ = flags.GetBool("unsafe")

	// conversation
	cfg.New, _ = flags.GetBool("new")
	cfg.MaxHistory, _ = flags.GetInt("max-history")
	cfg.MaxSpan, _ = flags.GetInt("max-span")

	// LLM
	internal.ParseLLM(cfg)
	// cfg.Models, _ = flags.GetString("models")
	// cfg.LLM.Provider, _ = flags.GetString("provider")
	// cfg.LLM.ApiKey, _ = flags.GetString("api-key")
	// cfg.LLM.Model, _ = flags.GetString("model")
	// cfg.LLM.BaseUrl, _ = flags.GetString("base-url")

	// // TODO redesign to simlify the model handling
	// // use the same model for all levels for now
	// var m = &model.Model{}
	// if cfg.LLM.Models == nil {
	// 	cfg.LLM.Models = make(map[model.Level]*model.Model)
	// }
	// cfg.LLM.Models[model.L1] = m
	// cfg.LLM.Models[model.L2] = m
	// cfg.LLM.Models[model.L3] = m

	// // TTS
	// cfg.TTS.Provider, _ = flags.GetString("tts-provider")
	// cfg.TTS.ApiKey, _ = flags.GetString("tts-api-key")
	// cfg.TTS.Model, _ = flags.GetString("tts-model")
	// cfg.TTS.BaseUrl, _ = flags.GetString("tts-base-url")
	//
	cfg.MaxTurns, _ = flags.GetInt("max-turns")
	cfg.MaxTime, _ = flags.GetInt("max-time")
}
