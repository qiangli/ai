package agent

import (
	"context"
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/internal/util"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/log"
)

func ProcessBashScript(ctx context.Context, cfg *api.AppConfig, script string) error {
	lines := strings.Split(script, "\n")
	if len(lines) > 1 {
		return confirmRun(
			ctx,
			cfg,
			"Run, edit, copy? [y/e/c/N] ",
			[]string{"yes", "edit", "copy", "no"},
			"no",
			script,
		)
	} else {
		return confirmRun(
			ctx,
			cfg,
			"Run? [y/N] ",
			[]string{"yes", "no"},
			"no",
			script,
		)
	}
}

func confirmRun(ctx context.Context, cfg *api.AppConfig, ps string, choices []string, defaultChoice, script string) error {
	answer, err := util.Confirm(ctx, ps, choices, defaultChoice, os.Stdin)
	if err != nil {
		return err
	}
	switch answer {
	case "edit":
		return editScript(ctx, cfg, script)
	case "copy":
		return copyScriptToClipboard(cfg, script)
	case "yes":
		return runScript(ctx, cfg, script)
	case "no":
	default:
	}
	return nil
}

func runScript(ctx context.Context, cfg *api.AppConfig, script string) error {
	log.GetLogger(ctx).Debugf("Running script:\n%s\n", script)

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

	// log.GetLogger(ctx).Debugf("Working directory: %s\n", wd)
	log.GetLogger(ctx).Debugf("Script file: %s\n", tmpFile.Name())

	cmd := exec.Command("bash", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = os.TempDir()

	return cmd.Run()
}

func copyScriptToClipboard(_ *api.AppConfig, script string) error {
	return util.NewClipboard().Write(script)
}

func editScript(ctx context.Context, cfg *api.AppConfig, script string) error {
	// editor := cfg.Editor
	editor := "vi"

	log.GetLogger(ctx).Debugf("Using editor: %s\n", editor)

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
