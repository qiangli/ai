package shell

import (
	"fmt"
	"strconv"
	"strings"
)

//https://pkg.go.dev/github.com/c-bata/go-prompt@v0.2.6#readme-features
// Key Binding	Description
// Ctrl + A	Go to the beginning of the line (Home)
// Ctrl + E	Go to the end of the line (End)
// Ctrl + P	Previous command (Up arrow)
// Ctrl + N	Next command (Down arrow)
// Ctrl + F	Forward one character
// Ctrl + B	Backward one character
// Ctrl + D	Delete character under the cursor
// Ctrl + H	Delete character before the cursor (Backspace)
// Ctrl + W	Cut the word before the cursor to the clipboard
// Ctrl + K	Cut the line after the cursor to the clipboard
// Ctrl + U	Cut the line before the cursor to the clipboard
// Ctrl + L	Clear the screen

func showAiShellKeyBindings() {
	fmt.Println("  \033[0;32mKey Bindings:\033[0m")
	fmt.Println("  Ctrl + A\tGo to the beginning of the line (Home)")
	fmt.Println("  Ctrl + E\tGo to the end of the line (End)")
	fmt.Println("  Ctrl + P\tPrevious command (Up arrow)")
	fmt.Println("  Ctrl + N\tNext command (Down arrow)")
	fmt.Println("  Ctrl + F\tForward one character")
	fmt.Println("  Ctrl + B\tBackward one character")
	fmt.Println("  Ctrl + D\tDelete character under the cursor")
	fmt.Println("  Ctrl + H\tDelete character before the cursor (Backspace)")
	fmt.Println("  Ctrl + W\tCut the word before the cursor to the clipboard")
	fmt.Println("  Ctrl + K\tCut the line after the cursor to the clipboard")
	fmt.Println("  Ctrl + U\tCut the line before the cursor to the clipboard")
	fmt.Println("  Ctrl + L\tClear the screen")
}

// https://github.com/antonmedv/walk
// Key binding	Description
// arrows, hjkl	Move cursor
// shift + arrows	Jump to start/end
// enter	Enter directory
// backspace	Exit directory
// space	Toggle preview
// esc, q	Exit with cd
// ctrl + c	Exit without cd
// /	Fuzzy search
// d, delete	Delete file or dir
// y	yank current dir
// .	Hide hidden files

func showExploreKeyBindings() {
	fmt.Println("  \033[0;32mKey Bindings:\033[0m")
	fmt.Println("  ←↑↓→          Move cursor")
	fmt.Println("  Enter         Enter directory")
	fmt.Println("  Backspace     Exit directory")
	fmt.Println("  Space         Toggle preview")
	fmt.Println("  Esc, q        Exit")
	fmt.Println("  Ctrl + C      Exit without cd")
	fmt.Println("  /             Fuzzy search")
	// fmt.Println("  y             Copy to clipboard")
	// fmt.Println("  .             Hide hidden files")
}

var builtin = []string{
	"help",
	"exit",
	"history",
	"alias",
	"env",
	"source",
	"explore",
}

func help(s string) {
	cmdArgs := strings.SplitN(s, " ", 2)
	var cmd, args string
	if len(cmdArgs) > 0 {
		cmd = cmdArgs[0]
	}
	if len(cmdArgs) > 1 {
		args = cmdArgs[1]
	}
	if cmd != "" {
		switch {
		case strings.HasPrefix(cmd, "word"):
			parts := strings.Fields(args)
			n := 10
			if len(parts) > 1 {
				if i, err := strconv.Atoi(parts[1]); err == nil {
					n = i
				}
			}
			wordCompleter.Show(n)
			return
		case strings.HasPrefix(cmd, "key"):
			showAiShellKeyBindings()
			return
		default:
			// fall through to the default help
		}
	}

	// default help
	var items = []struct {
		name        string
		description string
	}{
		{"exit", "exit ai shell"},
		{"history [-c]", "display or clear command history"},
		{"alias [name[=value]", "set or print aliases"},
		{"env [name[=value]", "export or print environment"},
		{"source [file]", "set alias and environment from file"},
		{"explore [--help] [path]", "explore local file system"},
		{"help", "help for ai shell"},
		{"@[agent]", "agent command"},
		{"/[command]", "slash (shell agent) command"},
	}

	width := 0
	for _, item := range items {
		if len(item.name) > width {
			width = len(item.name)
		}
	}

	for _, item := range items {
		fmt.Printf("  \033[0;32m%-*s\033[0m  │  %s\n", width, item.name, item.description)
	}
	fmt.Println()
	showAiShellKeyBindings()
}
