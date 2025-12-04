package agent

import (
	"context"
	// "os"
	// "strings"

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

	// // Bind the flags to viper using underscores
	// flags.VisitAll(func(f *pflag.Flag) {
	// 	key := strings.ReplaceAll(f.Name, "-", "_")
	// 	viper.BindPFlag(key, f)
	// })
}

// func run(cmd *cobra.Command, args []string) error {
// 	var ctx = context.TODO()

// 	cfg, err := setupAppConfig(ctx, args)
// 	if err != nil {
// 		return err
// 	}

// 	if err := internal.Validate(cfg); err != nil {
// 		return err
// 	}

// 	// watch mode
// 	// if cfg.Watch {
// 	// 	if err := watch.WatchRepo(ctx, cfg); err != nil {
// 	// 		log.GetLogger(ctx).Errorf("%v\n", err)
// 	// 	}
// 	// 	return nil
// 	// }
// 	// if cfg.ClipWatch {
// 	// 	if err := watch.WatchClipboard(ctx, cfg); err != nil {
// 	// 		log.GetLogger(ctx).Errorf("%v\n", err)
// 	// 	}
// 	// 	return nil
// 	// }

// 	// interactive mode
// 	// $ ai -i or $ ai --interactive
// 	if cfg.Interactive() {
// 		// // bubble
// 		// if len(cfg.Args) > 0 && cfg.Args[0] == "bubble" {
// 		// 	// _, err := bubble.BubbleGum(cfg.Args[1:])
// 		// 	if len(cfg.Args) < 3 {
// 		// 		bubble.Help()
// 		// 		return nil
// 		// 	}
// 		// 	sub := cfg.Args[1]
// 		// 	prompt := cfg.Args[2]

// 		// 	var err error
// 		// 	var result string
// 		// 	switch sub {
// 		// 	case "confirm":
// 		// 		result, err = bubble.Confirm(prompt)
// 		// 	case "choose":
// 		// 		if len(cfg.Args) < 5 {
// 		// 			log.GetLogger(ctx).Errorf("%v\n", "Not enough args")
// 		// 			return nil
// 		// 		}
// 		// 		multi, _ := strconv.ParseBool(cfg.Args[4])
// 		// 		result, err = bubble.Choose(prompt, cfg.Args[4:], multi)
// 		// 	case "file":
// 		// 		var p string
// 		// 		if len(cfg.Args) > 3 {
// 		// 			p = cfg.Args[3]
// 		// 		}
// 		// 		result, err = bubble.PickFile(prompt, p)
// 		// 	case "write":
// 		// 		var placeholder, value string
// 		// 		if len(cfg.Args) > 3 {
// 		// 			placeholder = cfg.Args[3]
// 		// 		}
// 		// 		if len(cfg.Args) > 4 {
// 		// 			value = cfg.Args[4]
// 		// 		}
// 		// 		result, err = bubble.Write(prompt, placeholder, value)
// 		// 	case "edit":
// 		// 		var placeholder, value string
// 		// 		if len(cfg.Args) > 3 {
// 		// 			placeholder = cfg.Args[3]
// 		// 		}
// 		// 		if len(cfg.Args) > 4 {
// 		// 			value = cfg.Args[4]
// 		// 		}
// 		// 		result, err = bubble.Edit(prompt, placeholder, value)
// 		// 	}
// 		// 	if err != nil {
// 		// 		log.GetLogger(ctx).Errorf("%v\n", err)
// 		// 	}
// 		// 	log.GetLogger(ctx).Printf("%s\n", result)
// 		// 	return nil
// 		// }

// 		if err := shell.Shell(ctx, cfg); err != nil {
// 			log.GetLogger(ctx).Errorf("%v\n", err)
// 		}
// 		return nil
// 	}

// 	// $ ai
// 	// if no args and no input, show help
// 	if !cfg.HasInput() {
// 		if err := cmd.Help(); err != nil {
// 			return err
// 		}
// 	}

// 	if err := agent.RunAgent(ctx, cfg); err != nil {
// 		log.GetLogger(ctx).Errorf("%v\n", err)
// 		return nil
// 	}

// 	return nil
// }

func setupAppConfig(ctx context.Context, argv []string) (*api.AppConfig, error) {
	argm, err := conf.ParseActionArgs(argv)
	if err != nil {
		return nil, err
	}
	var cfg = &api.AppConfig{}
	err = internal.ParseConfig(argm, cfg, argv)
	if err != nil {
		return nil, err
	}
	// // save original for non action commands
	// argm["arguments"] = argv
	// cfg.Arguments = argm
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
	if !conf.IsAction(argv[0]) {
		// argm := make(map[string]any)
		cfg.Arguments["command"] = argv[0]
		if len(argv) > 1 {
			cfg.Arguments["arguments"] = argv[1:]
		}
		if err := agent.RunSwarm(ctx, cfg); err != nil {
			log.GetLogger(ctx).Errorf("%v\n", err)
		}
		return nil
	}

	if err := agent.RunAgent(ctx, cfg); err != nil {
		log.GetLogger(ctx).Errorf("%v\n", err)
		return nil
	}
	return nil
}
