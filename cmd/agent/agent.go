package agent

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/qiangli/ai/agent"
	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/bubble"
	"github.com/qiangli/ai/internal/watch"
	"github.com/qiangli/ai/shell"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

var viper = internal.V

var AgentCmd = &cobra.Command{
	Use:                   "ai [OPTIONS] [@AGENT] MESSAGE...",
	Short:                 "AI Command Line Tool",
	Version:               internal.Version,
	DisableFlagsInUseLine: true,
	DisableSuggestions:    true,
	Args:                  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return Run(cmd, args)
	},
}

func init() {
	defaultCfg := os.Getenv("AI_CONFIG")

	pflags := AgentCmd.PersistentFlags()
	pflags.String("config", defaultCfg, "config file")
	pflags.MarkHidden("config")

	//
	addAgentFlags(AgentCmd)

	flags := AgentCmd.Flags()
	flags.SortFlags = true

	AgentCmd.CompletionOptions.DisableDefaultCmd = true

	AgentCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()
		err := Help(ctx, cmd, args)
		if err != nil {
			internal.Exit(ctx, err)
		}
	})

	// Bind the flags to viper using underscores
	flags.VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	// Bind the flags to viper using dots
	viper.BindPFlag("mcp.server_root", flags.Lookup("mcp-server-root"))

	viper.BindPFlag("sql.db_name", flags.Lookup("sql-db-name"))
	viper.BindPFlag("sql.db_host", flags.Lookup("sql-db-host"))
	viper.BindPFlag("sql.db_port", flags.Lookup("sql-db-port"))
	viper.BindPFlag("sql.db_username", flags.Lookup("sql-db-username"))
	viper.BindPFlag("sql.db_password", flags.Lookup("sql-db-password"))
}

func Run(cmd *cobra.Command, args []string) error {
	var ctx = context.TODO()

	cfg, err := setupAppConfig(ctx, args)
	if err != nil {
		return err
	}

	if err := internal.Validate(cfg); err != nil {
		return err
	}

	// watch mode
	if cfg.Watch {
		if err := watch.WatchRepo(ctx, cfg); err != nil {
			log.GetLogger(ctx).Errorf("%v\n", err)
		}
		return nil
	}
	if cfg.ClipWatch {
		if err := watch.WatchClipboard(ctx, cfg); err != nil {
			log.GetLogger(ctx).Errorf("%v\n", err)
		}
		return nil
	}

	// interactive mode
	// $ ai -i or $ ai --interactive
	if cfg.Interactive {
		// bubble
		if len(cfg.Args) > 0 && cfg.Args[0] == "bubble" {
			// _, err := bubble.BubbleGum(cfg.Args[1:])
			if len(cfg.Args) < 3 {
				bubble.Help()
				return nil
			}
			sub := cfg.Args[1]
			prompt := cfg.Args[2]

			var err error
			var result string
			switch sub {
			case "confirm":
				result, err = bubble.Confirm(prompt)
			case "choose":
				if len(cfg.Args) < 5 {
					log.GetLogger(ctx).Errorf("%v\n", "Not enough args")
					return nil
				}
				multi, _ := strconv.ParseBool(cfg.Args[4])
				result, err = bubble.Choose(prompt, cfg.Args[4:], multi)
			case "file":
				var p string
				if len(cfg.Args) > 3 {
					p = cfg.Args[3]
				}
				result, err = bubble.PickFile(prompt, p)
			case "write":
				var placeholder, value string
				if len(cfg.Args) > 3 {
					placeholder = cfg.Args[3]
				}
				if len(cfg.Args) > 4 {
					value = cfg.Args[4]
				}
				result, err = bubble.Write(prompt, placeholder, value)
			case "edit":
				var placeholder, value string
				if len(cfg.Args) > 3 {
					placeholder = cfg.Args[3]
				}
				if len(cfg.Args) > 4 {
					value = cfg.Args[4]
				}
				result, err = bubble.Edit(prompt, placeholder, value)
			}
			if err != nil {
				log.GetLogger(ctx).Errorf("%v\n", err)
			}
			log.GetLogger(ctx).Printf("%s\n", result)
			return nil
		}

		if err := shell.Shell(ctx, cfg); err != nil {
			log.GetLogger(ctx).Errorf("%v\n", err)
		}
		return nil
	}

	// $ ai
	// if no args and no input, show help
	if !cfg.HasInput() && !cfg.IsSpecial() && !cfg.Editing {
		if err := cmd.Help(); err != nil {
			return err
		}
	}

	if err := agent.RunAgent(ctx, cfg); err != nil {
		log.GetLogger(ctx).Errorf("%v\n", err)
	}

	return nil
}

// func setLogLevel(ctx context.Context, app *api.AppConfig) {
// 	if app.Quiet {
// 		log.GetLogger(ctx).SetLogLevel(log.Quiet)
// 		return
// 	}

// 	if app.Trace {
// 		log.GetLogger(ctx).SetLogLevel(log.Tracing)
// 		return
// 	}

// 	if app.Debug {
// 		log.GetLogger(ctx).SetLogLevel(log.Verbose)
// 	}
// }

// func setLogOutput(ctx context.Context, path string) (*log.FileWriter, error) {
// 	if path != "" {
// 		f, err := log.NewFileWriter(path)
// 		if err != nil {
// 			return nil, err
// 		}
// 		log.GetLogger(ctx).SetLogOutput(f)
// 		return f, nil
// 	}
// 	return nil, nil
// }

func setupAppConfig(ctx context.Context, args []string) (*api.AppConfig, error) {
	var cfg = &api.AppConfig{}
	err := internal.ParseConfig(viper, cfg, args)
	if err != nil {
		return nil, err
	}
	log.GetLogger(ctx).SetLogLevel(cfg.LogLevel)
	log.GetLogger(ctx).Debugf("Config: %+v\n", cfg)

	// fileLog, err := setLogOutput(ctx, cfg.Log)
	// if err != nil {
	// 	return nil, err
	// }
	// defer func() {
	// 	if fileLog != nil {
	// 		fileLog.Close()
	// 	}
	// }()
	// setLogLevel(ctx, cfg)

	return cfg, nil
}
