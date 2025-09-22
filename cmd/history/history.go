package history

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/shell"
	"github.com/qiangli/ai/swarm/api"
)

var viper = internal.V

var HistoryCmd = &cobra.Command{
	Use:                   "history",
	Short:                 "Show AI conversion history",
	DisableFlagsInUseLine: true,
	DisableSuggestions:    true,
	Run: func(cmd *cobra.Command, args []string) {
		var ctx = context.TODO()
		var cfg = &api.AppConfig{}

		if err := internal.ParseConfig(viper, cfg, args); err != nil {
			internal.Exit(ctx, err)
		}
		if err := historyConfig(ctx, cfg); err != nil {
			internal.Exit(ctx, err)
		}
	},
}

var flagReverse bool

func init() {
	flags := HistoryCmd.Flags()
	flags.BoolVarP(&flagReverse, "reverse", "r", false, "Sort by timestamp in reverse order")

	flags.SortFlags = true
	HistoryCmd.CompletionOptions.DisableDefaultCmd = true
}

func historyConfig(ctx context.Context, cfg *api.AppConfig) error {
	if _, err := os.Stat(cfg.ConfigFile); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("Configuration not found")
	}
	base := filepath.Dir(cfg.ConfigFile)
	histRoot := filepath.Join(base, "chat")
	var args = []string{"--sort", "time"}
	if flagReverse {
		args = append(args, "--reverse")
	}
	args = append(args, histRoot)

	return shell.Explore(ctx, args)
}
