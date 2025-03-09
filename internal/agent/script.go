package agent

import (
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

func ProcessBashScript(cfg *internal.AppConfig, script string) error {
	lines := strings.Split(script, "\n")
	if len(lines) > 1 {
		return confirmRun(
			cfg,
			"Run, edit, copy? [y/e/c/N] ",
			[]string{"yes", "edit", "copy", "no"},
			"no",
			script,
		)
	} else {
		return confirmRun(
			cfg,
			"Run? [y/N] ",
			[]string{"yes", "no"},
			"no",
			script,
		)
	}
}

func confirmRun(cfg *internal.AppConfig, ps string, choices []string, defaultChoice, script string) error {
	answer, err := util.Confirm(ps, choices, defaultChoice, os.Stdin)
	if err != nil {
		return err
	}
	switch answer {
	case "edit":
		return editScript(cfg, script)
	case "copy":
		return copyScriptToClipboard(cfg, script)
	case "yes":
		return runScript(cfg, script)
	case "no":
	default:
	}
	return nil
}

func runScript(cfg *internal.AppConfig, script string) error {
	log.Debugf("Running script:\n%s\n", script)

	tmpFile, err := os.CreateTemp("", "ai-script-*.sh")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(script)); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		return err
	}

	wd := cfg.WorkDir

	log.Debugf("Working directory: %s\n", wd)
	log.Debugf("Script file: %s\n", tmpFile.Name())

	cmd := exec.Command("bash", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = wd

	return cmd.Run()
}

func copyScriptToClipboard(_ *internal.AppConfig, script string) error {
	return util.NewClipboard().Write(script)
}

func editScript(cfg *internal.AppConfig, script string) error {
	editor := cfg.Editor

	log.Debugf("Using editor: %s\n", editor)

	tmpFile, err := os.CreateTemp("", "ai-script-*.sh")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(script)); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	cmd := exec.Command(editor, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
