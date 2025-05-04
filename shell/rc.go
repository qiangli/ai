package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-shellwords"
)

var rcFileMap = map[string]string{
	"bash": "~/.bashrc",
	"sh":   "~/.profile",
}

// listAlias executes the shell command to get the aliases
func listAlias(shellBin string) (map[string]string, error) {
	rc, ok := rcFileMap[filepath.Base(shellBin)]
	if !ok {
		return nil, fmt.Errorf("unsupported shell: %s", shellBin)
	}
	return aliasFile(shellBin, rc)
}

func aliasFile(shellBin string, file string) (map[string]string, error) {
	bin := fmt.Sprintf("source %s; alias", file)
	out, err := exec.Command(shellBin, "-c", bin).CombinedOutput()
	if err != nil {
		return nil, err
	}

	aliases := make(map[string]string)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line != "" {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 && strings.HasPrefix(parts[0], "alias ") {
				parts[0] = strings.TrimPrefix(parts[0], "alias ")
				parts[0] = strings.TrimSpace(parts[0])

				value := parts[1]
				parser := shellwords.NewParser()
				if args, err := parser.Parse(value); err == nil {
					value = strings.Join(args, " ")
				}
				aliases[parts[0]] = value
			}
		}
	}
	return aliases, nil
}

func sourceFile(shellBin string, rc string) error {
	envs, err := envFile(shellBin, rc)
	if err != nil {
		return err
	}
	for k, v := range envs {
		os.Setenv(k, v)
	}
	return nil
}

func env(shellBin string) (map[string]string, error) {
	rc, ok := rcFileMap[filepath.Base(shellBin)]
	if !ok {
		return nil, fmt.Errorf("unsupported shell: %s", shellBin)
	}

	return envFile(shellBin, rc)
}

func envFile(shellBin string, file string) (map[string]string, error) {
	bin := fmt.Sprintf("source %s; env", file)
	out, err := exec.Command(shellBin, "-c", bin).CombinedOutput()
	if err != nil {
		return nil, err
	}

	envs := make(map[string]string)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line != "" {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				envs[parts[0]] = parts[1]
			}
		}
	}
	return envs, nil
}

func runSource(shellBin string, args string) error {
	var file string
	if args != "" {
		file = args
	} else {
		var ok bool
		file, ok = rcFileMap[filepath.Base(shellBin)]
		if !ok {
			return fmt.Errorf("unsupported shell: %s", shellBin)
		}
	}

	file = subst(file)
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("file %s not found", file)
	}

	// update aliases
	aliases, _ := aliasFile(shellBin, file)
	for k, v := range aliases {
		aliasRegistry[k] = v
	}

	return sourceFile(shellBin, file)
}
