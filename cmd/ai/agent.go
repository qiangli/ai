package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/swarm/api"

	"github.com/qiangli/ai/agent"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/watch"
	"github.com/qiangli/ai/shell"
	"github.com/qiangli/ai/shell/edit"
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

	viper.BindPFlag("mcp.server-url", flags.Lookup("mcp-server-url"))

	viper.BindPFlag("sql.db-name", flags.Lookup("sql-db-name"))
	viper.BindPFlag("sql.db-host", flags.Lookup("sql-db-host"))
	viper.BindPFlag("sql.db-port", flags.Lookup("sql-db-port"))
	viper.BindPFlag("sql.db-username", flags.Lookup("sql-db-username"))
	viper.BindPFlag("sql.db-password", flags.Lookup("sql-db-password"))

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

	// watch
	if cfg.Watch {
		if err := watch.WatchRepo(cfg); err != nil {
			log.Errorln(err)
		}
		return nil
	}

	// interactive mode
	// $ ai -i or $ ai --interactive
	// TODO: implement interactive mode
	if cfg.Interactive {
		return shell.Shell(vars)
		// return fmt.Errorf("interactive mode not implemented yet")
	}

	// editing
	if cfg.Editing {
		s := cfg.GetQuery()
		s, err := openEditor(s)
		if err != nil {
			return err
		}
		//
		cfg.Message = s
		if err := agent.RunAgent(cfg); err != nil {
			return err
		}
		return nil
	}

	// $ ai
	// if no args and no input, show help
	if !cfg.HasInput() && !cfg.IsSpecial() {
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

func openEditor(s string) (string, error) {
	tmpfile, err := os.CreateTemp("", "*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpfile.Name())

	lines := splitByLength(s, 75)
	if _, err := tmpfile.WriteString(strings.Join(lines, "\n")); err != nil {
		tmpfile.Close()
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		return "", err
	}

	//
	if err := edit.Edit([]string{tmpfile.Name()}); err != nil {
		return "", err
	}

	edited, err := os.ReadFile(tmpfile.Name())
	if err != nil {
		return "", err
	}

	return string(edited), nil
}

// split s into length of around 80 char delimited by space
func splitByLength(s string, limit int) []string {
	words := strings.Fields(s)
	var lines []string
	var buf strings.Builder

	for _, word := range words {
		// If adding this word would exceed the limit, start a new line
		if buf.Len() > 0 && buf.Len()+len(word)+1 > limit {
			lines = append(lines, buf.String())
			buf.Reset()
		}
		if buf.Len() > 0 {
			buf.WriteByte(' ')
		}
		buf.WriteString(word)
	}
	// Add the last line if any
	if buf.Len() > 0 {
		lines = append(lines, buf.String())
	}
	return lines
}
