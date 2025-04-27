package shell

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/c-bata/go-prompt"
)

var filePathCompleter = FilePathCompleter{
	IgnoreCase: true,
}

// unique remove prompt suggestion duplicates
func unique(s []prompt.Suggest) []prompt.Suggest {
	keys := make(map[prompt.Suggest]bool)
	list := []prompt.Suggest{}
	for _, entry := range s {
		if _, ok := keys[entry]; !ok {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func Completer(d prompt.Document) []prompt.Suggest {
	var builtin = []string{
		"help",
		"exit",
		"history",
		"clear",
		"alias",
		"env",
		"source",
	}

	var s []prompt.Suggest

	for _, cmd := range builtin {
		s = append(s, prompt.Suggest{Text: cmd, Description: "ai shell"})
	}

	hist := getCommandHist()
	for _, command := range hist {
		s = append(s, prompt.Suggest{Text: command, Description: "history"})
	}

	for k := range commandRegistry {
		s = append(s, prompt.Suggest{Text: k, Description: "command"})
	}
	for k := range agentRegistry {
		s = append(s, prompt.Suggest{Text: "@" + k, Description: "agent"})
	}
	for k := range aliasRegistry {
		s = append(s, prompt.Suggest{Text: k, Description: "alias"})
	}
	s = prompt.FilterHasPrefix(unique(s), d.CurrentLine(), true)

	var isCd = IsCdCmd(d)

	// only show visited paths for cd command
	var visited []prompt.Suggest
	var files []prompt.Suggest

	if isCd {
		for _, k := range visitedRegistry.List() {
			visited = append(visited, prompt.Suggest{Text: k, Description: "cd"})
		}
	}

	var w = d.GetWordBeforeCursor()
	// only partially match the last word (sub dir)
	if len(w) > 0 {
		if strings.HasSuffix(w, string(os.PathSeparator)) {
			w = ""
		} else {
			w = filepath.Base(w)
		}
		w = strings.ToLower(w)
	}
	filter := func(fi os.DirEntry) bool {
		// only show directories for cd command
		if isCd && !fi.IsDir() {
			return false
		}
		if w == "" {
			return true
		}
		return strings.Contains(strings.ToLower(fi.Name()), w)
	}
	files = filePathCompleter.Complete(d, filter)

	completions := slices.Concat(files, visited, s)
	return completions
}

func IsCdCmd(pd prompt.Document) bool {
	cmdArgs := strings.Fields(pd.CurrentLineBeforeCursor())
	if len(cmdArgs) > 0 {
		return slices.Contains([]string{"cd"}, cmdArgs[0])
	}
	return false
}
