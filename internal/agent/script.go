package agent

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/qiangli/ai/internal/cb"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
)

func ProcessBashScript(cfg *llm.Config, script string) error {
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

func confirm(ps string, choices []string, defaultChoice string, in io.Reader) (string, error) {
	memo := make(map[string]string)
	for _, v := range choices {
		choice := strings.ToLower(v)
		memo[choice] = choice
		memo[choice[:1]] = choice
	}

	reader := bufio.NewReader(in)
	for {
		log.Promptf(ps)

		resp, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		resp = strings.ToLower(strings.TrimSpace(resp))
		if resp == "" {
			return defaultChoice, nil
		}
		result, ok := memo[resp]
		if ok {
			return result, nil
		}
	}
}

func confirmRun(cfg *llm.Config, ps string, choices []string, defaultChoice, script string) error {
	answer, err := confirm(ps, choices, defaultChoice, os.Stdin)
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

func runScript(cfg *llm.Config, script string) error {
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

func copyScriptToClipboard(_ *llm.Config, script string) error {
	return cb.NewClipboard().Write(script)
}

func editScript(cfg *llm.Config, script string) error {
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
