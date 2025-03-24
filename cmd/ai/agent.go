package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/internal/log"
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
		Help(cmd, args)
	})

	// Bind the flags to viper using underscores
	AgentCmd.Flags().VisitAll(func(f *pflag.Flag) {
		key := strings.ReplaceAll(f.Name, "-", "_")
		viper.BindPFlag(key, f)
	})

	// Bind the flags to viper using dots
	flags := AgentCmd.Flags()
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
	setLogLevel()

	fileLog, err := setLogOutput()
	if err != nil {
		return err
	}
	defer func() {
		if fileLog != nil {
			fileLog.Close()
		}
	}()

	cfg, err := internal.ParseConfig(args)
	if err != nil {
		return err
	}

	log.Debugf("Config: %+v %+v %+v\n", cfg, cfg.LLM, cfg.Db)

	// interactive mode
	// $ ai -i or $ ai --interactive
	// TODO: implement interactive mode
	if cfg.Interactive {
		// return shell.Bash(cfg)
		return fmt.Errorf("interactive mode not implemented yet")
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

func setLogLevel() {
	quiet := viper.GetBool("quiet")
	if quiet {
		log.SetLogLevel(log.Quiet)
		return
	}
	debug := viper.GetBool("verbose")
	if debug {
		log.SetLogLevel(log.Verbose)
	}

	// trace
	log.Trace = viper.GetBool("trace")
}

func setLogOutput() (*log.FileWriter, error) {
	pathname := viper.GetString("log")
	if pathname != "" {
		f, err := log.NewFileWriter(pathname)
		if err != nil {
			return nil, err
		}
		log.SetLogOutput(f)
		return f, nil
	}
	return nil, nil
}
