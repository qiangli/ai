package agent

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	// "strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	// "github.com/spf13/pflag"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
	// "github.com/qiangli/ai/shell"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

// var viper = internal.V

var AgentCmd = &cobra.Command{
	Use:                   "ai [OPTIONS] [@AGENT] MESSAGE...",
	Short:                 "AI Command Line Tool",
	Version:               internal.Version,
	DisableFlagsInUseLine: true,
	DisableSuggestions:    true,
	Args:                  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// return Run(cmd, args)
		return nil
	},
}

func init() {
	// defaultCfg := os.Getenv("AI_CONFIG")

	// pflags := AgentCmd.PersistentFlags()
	// pflags.String("config", defaultCfg, "config file")
	// pflags.MarkHidden("config")

	//
	// addAgentFlags(AgentCmd)

	flags := AgentCmd.Flags()
	flags.SortFlags = true

	AgentCmd.CompletionOptions.DisableDefaultCmd = true

	AgentCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()
		err := Help(ctx, args)
		if err != nil {
			internal.Exit(ctx, err)
		}
	})

}

func setupAppConfig(ctx context.Context, argv []string) (*api.AppConfig, error) {
	argm, err := conf.ParseActionArgs(argv)
	if err != nil {
		return nil, err
	}
	var cfg = &api.AppConfig{}
	err = ParseConfig(ctx, cfg, argv)
	if err != nil {
		return nil, err
	}
	cfg.Arguments = argm

	level := api.ToLogLevel(cfg.LogLevel)
	log.GetLogger(ctx).SetLogLevel(level)
	log.GetLogger(ctx).Debugf("Config: %+v\n", cfg)

	return cfg, nil
}

func Run(ctx context.Context, argv []string) error {

	cfg, err := setupAppConfig(ctx, argv)
	if err != nil {
		return err
	}

	// call local system command as tool:
	// sh:bash command arguments
	if !conf.IsAction(argv[0]) {
		args := make(map[string]any)
		args["kit"] = "sh"
		args["name"] = "bash"
		args["command"] = argv[0]
		if len(argv) > 1 {
			args["arguments"] = argv[1:]
		}
		maps.Copy(cfg.Arguments, args)

		if err := agent.RunSwarm(ctx, cfg); err != nil {
			log.GetLogger(ctx).Errorf("%v\n", err)
		}
		return nil
	}

	if err := agent.RunSwarm(ctx, cfg); err != nil {
		log.GetLogger(ctx).Errorf("%v\n", err)
		return nil
	}
	return nil
}

func ParseConfig(ctx context.Context, app *api.AppConfig, args []string) error {
	// defaults
	app.Format = "markdown"
	app.LogLevel = "info"

	app.Session = uuid.NewString()

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	app.Base = filepath.Join(home, ".ai")

	ws := filepath.Join(app.Base, "workspace")
	if v, err := internal.EnsureWorkspace(ws); err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	} else {
		app.Workspace = v
	}

	// stdin//pasteboard
	internal.ParseSpecialChars(app, args)

	in, err := agent.GetUserInput(ctx, app)
	if err != nil {
		return err
	}
	app.Message = in.Message

	return nil
}
