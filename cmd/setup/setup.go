package setup

import (
	"context"
	// "errors"
	// "io/fs"
	"os"
	"os/exec"
	// "path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/swarm/api"
)

var viper = internal.V

var SetupCmd = &cobra.Command{
	Use:                   "setup",
	Short:                 "Set up AI configuration",
	DisableFlagsInUseLine: true,
	DisableSuggestions:    true,
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.TODO()
		var cfg = &api.AppConfig{}

		if err := internal.ParseConfig(viper, cfg, args); err != nil {
			internal.Exit(ctx, err)
		}
		if err := setupConfig(cfg); err != nil {
			internal.Exit(ctx, err)
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

	// if _, err := os.Stat(cfg.ConfigFile); errors.Is(err, os.ErrNotExist) {
	// 	base := filepath.Dir(cfg.ConfigFile)
	// 	if err := os.MkdirAll(base, os.ModePerm); err != nil {
	// 		return err
	// 	}
	// 	cfgData := internal.GetConfigData()
	// 	err = fs.WalkDir(cfgData, ".", func(relPath string, d fs.DirEntry, err error) error {
	// 		if err != nil {
	// 			return err
	// 		}

	// 		stripped := relPath
	// 		if strings.HasPrefix(relPath, "data/") {
	// 			stripped = strings.TrimPrefix(relPath, "data/")
	// 		}
	// 		fullPath := filepath.Join(base, stripped)

	// 		if d.IsDir() {
	// 			return nil
	// 		}

	// 		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
	// 			return err
	// 		}

	// 		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
	// 			data, err := cfgData.ReadFile(relPath)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			if err := os.WriteFile(fullPath, data, 0600); err != nil {
	// 				return err
	// 			}
	// 		}
	// 		return nil
	// 	})
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	//
	editor := "vi"
	// if editor == "" {
	// 	editor = internal.DefaultEditor
	// }
	// support simple args for editor command line
	cmdArgs := strings.Fields(editor)
	var bin string
	var args []string
	bin = cmdArgs[0]
	if len(cmdArgs) > 1 {
		args = cmdArgs[1:]
	}
	// args = append(args, cfg.ConfigFile)

	cmd := exec.Command(bin, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
