package agent

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/internal/log"
)

var AgentCmd = &cobra.Command{
	Use:   "ai [OPTIONS] AGENT [message...]",
	Short: "AI command line tool",
	RunE: func(cmd *cobra.Command, args []string) error {
		return Run(cmd, args)
	},
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

	// // $ ai
	// if cfg.Command == "" && len(cfg.Args) == 0 && cfg.Message == "" {
	// 	if !cfg.IsPiped && !cfg.Stdin {
	// 		return cmd.Help()
	// 	}
	// }

	if err := agent.HandleCommand(cfg); err != nil {
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

func init() {
	addFlags(AgentCmd)

	AgentCmd.Flags().SortFlags = true
	AgentCmd.CompletionOptions.DisableDefaultCmd = true
}
