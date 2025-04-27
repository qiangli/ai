package shell

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal/log"
)

func execCommand(shellBin, original string) error {
	log.Debugf("original command: %q\n", original)

	parsed, err := parseCommand(original)
	if err != nil {
		return err
	}
	log.Debugf("parsed command: %+v\n", parsed)

	// Handle special case for "cd" command
	// assume single argument for "cd" command
	// update PWD environment variable as needed
	if len(parsed) > 0 && parsed[0][0] == "cd" {
		if len(parsed[0]) > 1 {
			dir := parsed[0][1]
			dir = subst(dir)
			return Chdir(dir)
		}
		user, err := user.Current()
		if err != nil {
			return err
		}
		return Chdir(user.HomeDir)
	}

	var modified []string
	for _, part := range parsed {
		if len(part) == 1 && isSep(part[0]) {
			modified = append(modified, part[0])
		} else {
			modified = append(modified, strings.Join(part, " "))
		}
	}

	log.Debugf("modified command: %+v\n", modified)

	// Execute the command
	command := strings.Join(modified, " ")
	log.Debugf("Executing command: %q\n", command)

	cmd := exec.Command(shellBin, "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

var commandSeps = []string{">>", "<<", ">", "<", "&&", "&", "||", "|", ";"}

var isSep = func(s string) bool {
	return slices.Contains(commandSeps, s)
}

func parseCommand(original string) ([][]string, error) {
	// Check if the command needs expansion
	// internal commands need expansion unlike external commands which is taken care of by the shell
	needsExpand := func(s string) bool {
		return slices.Contains([]string{"cd"}, s)
	}

	expand := func(sub string) ([]string, error) {
		args := strings.Fields(sub)

		if len(args) == 0 {
			return args, nil
		}

		if needsExpand(args[0]) {
			if err := substAll(args); err != nil {
				return nil, err
			}
		}

		// remove empty arguments
		var filtered []string
		for _, arg := range args {
			arg = strings.TrimSpace(arg)
			if arg != "" {
				filtered = append(filtered, arg)
			}
		}
		return filtered, nil
	}

	// Split the command string by the separators
	// and recursively expand each part.
	// This is a recursive function to handle nested commands.
	var split func(s string, seps []string) []string

	split = func(s string, seps []string) []string {
		log.Debugf("Splitting: %q with separators: %+v\n", s, seps)

		if len(seps) == 0 {
			return []string{s}
		}

		var result []string
		parts := strings.Split(s, seps[0])
		for i, part := range parts {
			if i > 0 {
				result = append(result, seps[0])
			}
			subs := split(part, seps[1:])
			result = append(result, subs...)
		}
		return result
	}

	// subcommands
	var subcommands = split(original, commandSeps)
	var expanded [][]string

	for _, sub := range subcommands {
		if isSep(sub) {
			expanded = append(expanded, []string{sub})
			continue
		}
		c, err := expand(sub)
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, c)
	}
	return expanded, nil
}

func substAll(args []string) error {
	user, err := user.Current()
	if err != nil {
		return err
	}

	// expand environment variables
	for i, arg := range args {
		if strings.Contains(arg, "$") {
			args[i] = os.ExpandEnv(arg)
		}
	}

	// expand tilde (~) for home directory
	for i, arg := range args {
		if strings.HasPrefix(arg, "~") {
			args[i] = strings.Replace(arg, "~", user.HomeDir, 1)
		}
	}

	// globbing
	for i, arg := range args {
		if strings.Contains(arg, "*") || strings.Contains(arg, "?") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				continue
			}
			if len(matches) > 0 {
				args[i] = strings.Join(matches, " ")
			} else {
				args[i] = arg
			}
		}
	}
	return nil
}

// subst replaces environment variables and expands tilde (~) in the given string.
func subst(s string) string {
	args := []string{s}
	if err := substAll(args); err != nil {
		return s
	}
	return args[0]
}

func runHistory(args string) error {
	if args == "-c" {
		clearHistory()
	} else {
		showHistory()
	}
	return nil
}

func runAlias(args string) error {
	if args == "" {
		showAlias()
		return nil
	}

	parts := strings.SplitN(args, "=", 2)
	if len(parts) == 1 {
		k := strings.TrimSpace(parts[0])
		fmt.Println("  \033[0;32m" + k + " \033[0m =  " + aliasRegistry[k])
	} else {
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if v != "" {
			v = subst(v)
		}
		aliasRegistry[k] = v
	}
	return nil
}

func showAlias() {
	var keys []string
	for k := range aliasRegistry {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	width := 0
	for _, k := range keys {
		if len(k) > width {
			width = len(k)
		}
	}

	for _, k := range keys {
		padded := fmt.Sprintf("%-*s", width, k)
		fmt.Println("  \033[0;32m" + padded + " \033[0m =  " + aliasRegistry[k])
	}
}

func runEnv(args string) error {
	if args == "" {
		showEnv()
		return nil
	}

	parts := strings.SplitN(args, "=", 2)
	if len(parts) == 1 {
		k := strings.TrimSpace(parts[0])
		v := os.Getenv(k)
		fmt.Println("  \033[0;32m" + k + "\033[0;33m=\033[0m" + v)
	} else {
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if v == "" {
			os.Unsetenv(k)
		} else {
			// special handling for PATH
			// treat each path as a separate entry
			if k == "PATH" {
				paths := strings.Split(v, string(os.PathListSeparator))
				if err := substAll(paths); err != nil {
					return err
				}
				v = strings.Join(paths, string(os.PathListSeparator))
			} else {
				v = subst(v)
			}
			os.Setenv(k, v)
		}
	}
	return nil
}

func showEnv() {
	envMap := make(map[string]string)
	for _, e := range os.Environ() {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	var keys []string
	for k := range envMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		fmt.Println("  \033[0;32m" + k + "\033[0;33m=\033[0m" + envMap[k])
	}
}
