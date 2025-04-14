package shell

import (
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/c-bata/go-prompt"
)

var filePathCompleter = FilePathCompleter{}

// unique remove prompt suggestion duplicates
func unique(intSlice []prompt.Suggest) []prompt.Suggest {
	keys := make(map[prompt.Suggest]bool)
	list := []prompt.Suggest{}
	for _, entry := range intSlice {
		if _, value := keys[entry]; !value {
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
	s := []prompt.Suggest{}
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
	for k := range visitedRegistry {
		s = append(s, prompt.Suggest{Text: k, Description: "visited"})
	}

	var w = d.GetWordBeforeCursor()
	// only match the last word (sub dir)
	if len(w) > 0 {
		if strings.HasSuffix(w, string(os.PathSeparator)) {
			w = ""
		} else {
			w = filepath.Base(w)
		}
		w = strings.ToLower(w)
	}

	var isCd = false
	isCmdDir := func(s string) bool {
		return slices.Contains([]string{"cd"}, s)
	}
	// TODO compound command handling
	cmdArgs := strings.Fields(d.CurrentLineBeforeCursor())
	if len(cmdArgs) > 0 {
		cmd := cmdArgs[0]
		if isCmdDir(cmd) {
			isCd = true
		} else if len(aliasRegistry) > 0 {
			for k, v := range aliasRegistry {
				if k == cmd {
					part := strings.Fields(v)
					if len(part) > 0 && isCmdDir(part[0]) {
						isCd = true
					}
				}
			}
		}
	}

	filePathCompleter.Filter = func(fi os.DirEntry) bool {
		if isCd && !fi.IsDir() {
			return false
		}
		if w == "" {
			return true
		}
		return strings.Contains(strings.ToLower(fi.Name()), w)
	}

	completions := filePathCompleter.Complete(d)
	completions = append(completions, prompt.FilterHasPrefix(unique(s), d.CurrentLine(), true)...)
	return completions
}

type FilePathCompleter struct {
	Filter func(fi os.DirEntry) bool
}

func (c *FilePathCompleter) Complete(pd prompt.Document) []prompt.Suggest {
	base := filepath.Clean(os.Getenv("PWD"))
	if abs, err := filepath.Abs(base); err == nil {
		base = abs
	}

	dir := base
	// resolve subdir
	w := pd.GetWordBeforeCursor()
	if w != "" {
		p := filepath.Join(base, w)
		if fi, err := os.Stat(p); err == nil {
			if fi.IsDir() {
				dir = p
			} else {
				dir = filepath.Dir(p)
			}
		} else {
			p = filepath.Dir(p)
			if fi, err := os.Stat(p); err == nil {
				if fi.IsDir() {
					dir = p
				}
			}
		}
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return []prompt.Suggest{}
	}

	suggests := make([]prompt.Suggest, 0, len(files))
	for _, f := range files {
		if c.Filter != nil && !c.Filter(f) {
			continue
		}
		var desc string
		if f.IsDir() {
			desc = "directory"
		} else {
			desc = "file"
		}
		suggests = append(suggests, prompt.Suggest{Text: f.Name(), Description: desc})
	}
	return suggests
}
