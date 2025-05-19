package setup

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/swarm/api"
)

var SetupCmd = &cobra.Command{
	Use:                   "setup",
	Short:                 "Set up AI configuration",
	DisableFlagsInUseLine: true,
	DisableSuggestions:    true,
	Run: func(cmd *cobra.Command, args []string) {
		cfg, err := internal.ParseConfig(args)
		if err != nil {
			internal.Exit(err)
		}
		if err := setupConfig(cfg); err != nil {
			internal.Exit(err)
		}
	},
}

func init() {
	flags := SetupCmd.Flags()
	flags.String("editor", "vi", "Specify editor to use")

	flags.SortFlags = true
	SetupCmd.CompletionOptions.DisableDefaultCmd = true
}

// Use the configFileContent variable in your application as needed
func setupConfig(cfg *api.AppConfig) error {
	if _, err := os.Stat(cfg.ConfigFile); errors.Is(err, os.ErrNotExist) {
		dir := filepath.Dir(cfg.ConfigFile)
		if err := os.MkdirAll(dir, os.ModePerm); err != nil {
			return err
		}
		content := internal.GetDefaultConfig()
		if err := os.WriteFile(cfg.ConfigFile, []byte(content), 0644); err != nil {
			return err
		}
	}

	//
	// support simple args for editor command line
	cmdArgs := strings.Fields(cfg.Editor)
	var bin string
	var args []string
	bin = cmdArgs[0]
	if len(cmdArgs) > 1 {
		args = cmdArgs[1:]
	}
	args = append(args, cfg.ConfigFile)

	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
