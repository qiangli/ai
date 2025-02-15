package agent

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/qiangli/ai/internal"
)

// Use the configFileContent variable in your application as needed
func setupConfig(cfg *internal.AppConfig) error {
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

	cmd := exec.Command(cfg.Editor, cfg.ConfigFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}
