package agent

import (
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/atm/conf"
	"github.com/qiangli/ai/swarm/log"
)

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

func setupAppConfig(ctx context.Context, app *api.AppConfig) error {
	app.Format = "markdown"
	app.LogLevel = "quiet"
	app.Session = uuid.NewString()

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	app.Base = filepath.Join(home, ".ai")

	ws := filepath.Join(app.Base, "workspace")
	if v, err := EnsureWorkspace(ws); err != nil {
		return fmt.Errorf("failed to resolve workspace: %w", err)
	} else {
		app.Workspace = v
	}

	level := api.ToLogLevel(app.LogLevel)
	log.GetLogger(ctx).SetLogLevel(level)
	log.GetLogger(ctx).Debugf("Config: %+v\n", app)

	return nil
}

func parseAppConfig(ctx context.Context, app *api.AppConfig, argv []string) error {
	argv = ParseSpecialChars(app, argv)
	argm, err := conf.ParseActionArgs(argv)
	if err != nil {
		return err
	}
	maps.Copy(app.Arguments, argm)

	in, err := agent.GetUserInput(ctx, app, api.ToString(argm["message"]))
	if err != nil {
		return err
	}
	app.Message = in.Message

	maps.Copy(app.Arguments, app.ToMap())
	return nil
}

func Run(ctx context.Context, argv []string) error {
	var app = &api.AppConfig{}
	app.Arguments = make(map[string]any)
	err := setupAppConfig(ctx, app)
	if err != nil {
		return err
	}

	if conf.IsAction(argv[0]) {
		err := parseAppConfig(ctx, app, argv)
		if err != nil {
			return err
		}
	} else if conf.IsSlash(argv[0]) {
		// call local system command as tool:
		// sh:bash command arguments
		app.Arguments["kit"] = "sh"
		app.Arguments["name"] = "bash"
		app.Arguments["command"] = argv[0]
		if len(argv) > 1 {
			app.Arguments["arguments"] = argv[1:]
		}
	} else {
		app.Arguments["message"] = strings.Join(argv, " ")
	}

	if err := agent.RunSwarm(ctx, app); err != nil {
		log.GetLogger(ctx).Errorf("%v\n", err)
		return nil
	}
	return nil
}
