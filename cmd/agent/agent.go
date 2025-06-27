package agent

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/swarm/api"

	"github.com/qiangli/ai/agent"
	"github.com/qiangli/ai/bubble"
	"github.com/qiangli/ai/internal/hub"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/watch"
	"github.com/qiangli/ai/shell"
	"github.com/qiangli/ai/swarm"
)

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
	// default: ~/.ai/config.yaml
	if defaultCfg == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			defaultCfg = filepath.Join(home, ".ai", "config.yaml")
		}
	}
	AgentCmd.PersistentFlags().StringVar(&internal.ConfigFile, "config", defaultCfg, "config file")

	//
	addAgentFlags(AgentCmd)
	AgentCmd.Flags().SortFlags = true
	AgentCmd.CompletionOptions.DisableDefaultCmd = true
	AgentCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		err := Help(cmd, args)
		if err != nil {
			internal.Exit(err)
		}
	})

	// Bind the flags to viper using underscores
	AgentCmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	// Bind the flags to viper using dots
	flags := AgentCmd.Flags()

	viper.BindPFlag("mcp.server_root", flags.Lookup("mcp-server-root"))

	viper.BindPFlag("sql.db_name", flags.Lookup("sql-db-name"))
	viper.BindPFlag("sql.db_host", flags.Lookup("sql-db-host"))
	viper.BindPFlag("sql.db_port", flags.Lookup("sql-db-port"))
	viper.BindPFlag("sql.db_username", flags.Lookup("sql-db-username"))
	viper.BindPFlag("sql.db_password", flags.Lookup("sql-db-password"))

	//
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ai")
	viper.BindEnv("api-key", "AI_API_KEY", "OPENAI_API_KEY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
}

func Run(cmd *cobra.Command, args []string) error {
	cfg, err := setupAppConfig(args)
	if err != nil {
		return err
	}

	vars, err := swarm.InitVars(cfg)
	if err != nil {
		return err
	}

	log.Debugf("Initialized variables: %+v\n", vars)

	// hub service
	if cfg.Hub {
		hub.StartServer(cfg)
		return nil
	}

	// watch mode
	if cfg.Watch {
		if err := watch.WatchRepo(cfg); err != nil {
			log.Errorln(err)
		}
		return nil
	}
	if cfg.ClipWatch {
		if err := watch.WatchClipboard(cfg); err != nil {
			log.Errorln(err)
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
					log.Errorln("Not enough args")
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
				log.Errorln(err)
			}
			log.Println(result)
			return nil
		}

		//
		if err := shell.Shell(vars); err != nil {
			log.Errorln(err)
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

	if err := agent.RunAgent(cfg); err != nil {
		log.Errorln(err)
	}

	return nil
}

func setLogLevel(app *api.AppConfig) {
	if app.Quiet {
		log.SetLogLevel(log.Quiet)
		return
	}
	if app.Debug {
		log.SetLogLevel(log.Verbose)
	}
}

func setLogOutput(path string) (*log.FileWriter, error) {
	if path != "" {
		f, err := log.NewFileWriter(path)
		if err != nil {
			return nil, err
		}
		log.SetLogOutput(f)
		return f, nil
	}
	return nil, nil
}

func setupAppConfig(args []string) (*api.AppConfig, error) {
	cfg, err := internal.ParseConfig(args)
	if err != nil {
		return nil, err
	}

	log.Debugf("Config: %+v %+v %+v\n", cfg, cfg.LLM, cfg.DBCred)

	fileLog, err := setLogOutput(cfg.Log)
	if err != nil {
		return nil, err
	}
	defer func() {
		if fileLog != nil {
			fileLog.Close()
		}
	}()
	setLogLevel(cfg)

	return cfg, nil
}
