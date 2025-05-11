package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
	"github.com/mattn/go-isatty"
	"golang.org/x/term"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

func Shell(vars *api.Vars) error {
	os.Setenv("SHELL", vars.Config.Shell)

	initRegistry(vars)

	initRc(vars)

	//
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	restoreState := func() {
		_ = term.Restore(int(os.Stdin.Fd()), oldState)
	}
	defer restoreState()

	// run sub commands
	if len(vars.Config.Args) > 0 {
		input := strings.Join(vars.Config.Args, " ")
		dispatch(vars.Config.Shell, input)
		return nil
	}

	// command loop
	if !interactive() {
		return nil
	}

	prompter, err := createPrompter()
	if err != nil {
		return err
	}

	for {
		prompter()

		input := prompt.Input(
			"",
			Completer,
			prompt.OptionHistory(getCommandHist()),
			prompt.OptionSuggestionBGColor(prompt.DefaultColor),
			prompt.OptionInputTextColor(prompt.Cyan),
			prompt.OptionMaxSuggestion(6),
			prompt.OptionTitle("ai"),
			prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator),
			prompt.OptionAddKeyBind(
				prompt.KeyBind{
					Key: prompt.ControlD,
					Fn: func(prompt_toolkit *prompt.Buffer) {
						prompter()
					},
				},
				prompt.KeyBind{
					Key: prompt.ControlC,
					Fn: func(_ *prompt.Buffer) {
						restoreState()
						os.Exit(0)
					},
				},
			),
			prompt.OptionPreviewSuggestionTextColor(prompt.DefaultColor),
			prompt.OptionScrollbarBGColor(prompt.DefaultColor),
		)

		if !dispatch(vars.Config.Shell, input) {
			return nil
		}
	}
}

func interactive() bool {
	return isatty.IsCygwinTerminal(os.Stdout.Fd()) || isatty.IsTerminal(os.Stdout.Fd())
}

func commandErr(command string, err error) {
	fmt.Printf("\033[31m✗\033[0m %s: %s\n", command, err.Error())
}

func initRc(vars *api.Vars) error {
	rc := filepath.Join(filepath.Dir(vars.Config.ConfigFile), "rc.sh")
	if _, err := os.Stat(rc); err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return nil
	}
	if err := runSource(vars.Config.Shell, rc); err != nil {
		return err
	}
	return nil
}

// dispatch returns true to continue, false to exit
func dispatch(shellBin, input string) bool {
	input = strings.Replace(input, "\n", "", -1)
	input = strings.TrimSpace(input)
	if input == "" {
		return true
	}

	log.Debugf("input command: %q\n", input)

	// cmd + args
	cmdArgs := strings.SplitN(input, " ", 2)

	// command args
	var command = cmdArgs[0]
	var args string
	if len(cmdArgs) > 1 {
		args = cmdArgs[1]
	}

	log.Debugf("command: %q, args: %q\n", command, args)

	// execute history command
	if strings.HasPrefix(command, "!") {
		if command == "!" {
			runHistory(args)
			return true
		}

		// history command
		num := command[1:]
		hist, err := getHistory(num)
		if err != nil {
			commandErr(command, err)
			return true
		}

		// fall through, not to continue
		// may need further processing for builtin/agent/alias commands
		log.Debugf("history command: %q, args: %q\n", hist, args)
		input = fmt.Sprintf("%s %s", hist, args)
		command = hist
		log.Debugf("proceed to run modified history !%s: command %q, input: %q\n", num, command, input)
	}

	// built-in commands:
	// help, history, exit
	if strings.Compare("help", command) == 0 {
		help(args)
		return true
	} else if strings.Compare("history", command) == 0 {
		runHistory(args)
		return true
	} else if strings.Compare("exit", command) == 0 {
		return false
	}

	// simulate builtin shell commands:
	// alias, source, env
	if strings.Compare("alias", command) == 0 {
		if err := runAlias(args); err != nil {
			commandErr(command, err)
		}
		updateHistory(input)
		return true
	} else if strings.Compare("source", command) == 0 ||
		strings.Compare(".", command) == 0 {
		if err := runSource(shellBin, args); err != nil {
			commandErr(command, err)
		}
		updateHistory(input)
		return true
	} else if strings.Compare("env", command) == 0 ||
		strings.Compare("export", command) == 0 {
		if err := runEnv(args); err != nil {
			commandErr(command, err)
		}
		updateHistory(input)
		return true
	} else if strings.Compare("edit", command) == 0 {
		if err := runEdit(args); err != nil {
			commandErr(command, err)
		}
		updateHistory(input)
		return true
	} else if strings.Compare("explore", command) == 0 {
		if err := runExplore(args); err != nil {
			commandErr(command, err)
		}
		updateHistory(input)
		return true
	}

	runShell := func(modified string, save bool) {
		// shell
		if err := execCommand(shellBin, modified, save); err != nil {
			commandErr(modified, err)
		}
		// update history with original command
		updateHistory(input)
	}

	// ai commands
	var special = []string{"/help", "/setup", "/mcp"}
	isAgent := func(s string) bool {
		return strings.HasPrefix(s, "@")
	}
	// NOTE full path system command will no longer work (directly)
	// e.g. /bin/ls will be: ai @shell/bin/ls ...
	// command may be interpreted and executed via tools call by LLM
	isSlash := func(s string) bool {
		return strings.HasPrefix(s, "/")
	}
	// alias commands
	isAlias := func(s string) bool {
		_, ok := aliasRegistry[s]
		return ok
	}

	// ai
	// /[cmd] ...
	// /help /setup /mcp
	// @[agent] ...
	// ... @[agent]
	makeAgentCmd := func(s string, parts []string) string {
		args := parts

		if args[0] == "ai" {
			args = args[1:]
		}
		if len(args) > 1 && isAgent(args[len(args)-1]) {
			args = args[:len(args)-1]
		}

		//
		for _, cmd := range special {
			if strings.Compare(cmd, s) == 0 {
				return fmt.Sprintf("ai %s %s", s, strings.Join(args, " "))
			}
		}

		var cmd string
		if isSlash(s) {
			cmd = fmt.Sprintf("ai @shell%s %s", s, strings.Join(args, " "))
		} else if isAgent(s) {
			cmd = fmt.Sprintf("ai %s %s", s, strings.Join(args, " "))
		}

		return cmd
	}

	runAgent := func(s string, parts []string) {
		cmd := makeAgentCmd(s, parts)
		runShell(cmd, true)
	}

	var modified = input

	// slash/agent commands
	parts := strings.Fields(input)
	switch len(parts) {
	case 0:
		// not reachable
		return true
	case 1:
		first := parts[0]
		if isSlash(first) || isAgent(first) {
			runAgent(first, parts)
			return true
		}
	default:
		first := parts[0]
		last := parts[len(parts)-1]
		if isAgent(last) {
			runAgent(last, parts)
			return true
		} else if isSlash(first) || isAgent(first) {
			runAgent(first, parts)
			return true
		}
	}

	if isAlias(parts[0]) {
		alias := aliasRegistry[parts[0]]
		if len(parts) > 1 {
			modified = alias + " " + strings.Join(parts[1:], " ")
		} else {
			modified = alias
		}
	}

	log.Debugf("modified command: %q\n", modified)

	if modified == "" {
		return true
	}

	runShell(modified, false)
	return true
}
