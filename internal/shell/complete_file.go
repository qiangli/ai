package shell

import (
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"github.com/c-bata/go-prompt"
)

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
