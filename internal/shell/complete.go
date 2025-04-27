package shell

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"
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
			visited = append(visited, prompt.Suggest{Text: k, Description: "visited"})
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

// https://github.com/c-bata/go-prompt/blob/master/completer/file.go
var (
	// FilePathCompletionSeparator holds separate characters.
	FilePathCompletionSeparator = string([]byte{' ', os.PathSeparator})
)

// FilePathCompleter is a completer for your local file system.
// Please caution that you need to set OptionCompletionWordSeparator(completer.FilePathCompletionSeparator)
// when you use this completer.
type FilePathCompleter struct {
	IgnoreCase bool
}

func cleanFilePath(path string) (dir, base string, err error) {
	var endsWithSeparator bool
	if len(path) >= 1 && path[len(path)-1] == os.PathSeparator {
		endsWithSeparator = true
	}

	if runtime.GOOS != "windows" && len(path) >= 2 && path[0:2] == "~/" {
		me, err := user.Current()
		if err != nil {
			return "", "", err
		}
		path = filepath.Join(me.HomeDir, path[1:])
	}
	path = filepath.Clean(os.ExpandEnv(path))
	dir = filepath.Dir(path)
	base = filepath.Base(path)

	if endsWithSeparator {
		dir = path + string(os.PathSeparator) // Append slash(in POSIX) if path ends with slash.
		base = ""                             // Set empty string if path ends with separator.
	}
	return dir, base, nil
}

// Complete returns suggestions from your local file system.
func (c *FilePathCompleter) Complete(d prompt.Document, filter func(os.DirEntry) bool) []prompt.Suggest {
	var err error
	var dir, base string

	w := d.GetWordBeforeCursor()
	if len(w) > 0 {
		dir, base, err = cleanFilePath(w)
		if err != nil {
			return nil
		}
	} else {
		dir = "."
	}

	files, err := os.ReadDir(dir)
	if err != nil && os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return nil
	}

	suggests := make([]prompt.Suggest, 0, len(files))
	for _, f := range files {
		if filter != nil && !filter(f) {
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

	return prompt.FilterHasPrefix(suggests, base, c.IgnoreCase)
}
