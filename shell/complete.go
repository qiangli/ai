package shell

import (
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/c-bata/go-prompt"
	"github.com/kballard/go-shellquote"
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

	filter := func(fi os.DirEntry) bool {
		// only show directories for cd command
		if isCd && !fi.IsDir() {
			return false
		}
		// ignore hidden files
		if fi.Name()[0] == '.' {
			return false
		}
		return true
	}
	files = filePathCompleter.Complete(d, filter)

	words := wordCompleter.Complete(d)

	completions := slices.Concat(files, visited, s, words)
	return completions
}

func IsCdCmd(pd prompt.Document) bool {
	cmdArgs := strings.Fields(pd.CurrentLineBeforeCursor())
	if len(cmdArgs) > 0 {
		return slices.Contains([]string{"cd"}, cmdArgs[0])
	}
	return false
}

var wordCompleter = NewWordCompleter(100, 3)

func (wc *WordCompleter) Complete(d prompt.Document) []prompt.Suggest {
	const maxN = 5

	word := d.GetWordBeforeCursor()
	if wc.ignoreCase {
		word = strings.ToUpper(word)
	}
	suggestions := make([]prompt.Suggest, 0)
	for _, wf := range wc.counter.Suggest(word, maxN) {
		suggestions = append(suggestions, prompt.Suggest{Text: wf.Word, Description: "word"})
	}
	return suggestions
}

type WordCompleter struct {
	ignoreCase bool

	counter *WordCounter
	headN   int
	tailM   int

	headCount int
	tailBuf   []string
	tailPos   int

	mu sync.Mutex
}

func NewWordCompleter(headN, tailM int) *WordCompleter {
	return &WordCompleter{
		counter:    DefaultWordCounter(),
		headN:      headN,
		tailM:      tailM,
		headCount:  0,
		tailPos:    0,
		tailBuf:    make([]string, tailM),
		ignoreCase: true,
	}
}

// Capture captures lines; processes first N synchronously,
// buffers last M, and processes buffer when which == 99
func (wc *WordCompleter) Capture(which int, line string) error {
	const minLength = 7

	if which == 99 {
		wc.mu.Lock()
		defer wc.mu.Unlock()

		// process tail buffer in order
		numTail := wc.tailM
		if wc.headCount > wc.headN {
			if wc.headCount-wc.headN < wc.tailM {
				numTail = wc.headCount - wc.headN
			}
		} else {
			numTail = 0
		}
		start := wc.tailPos
		for i := 0; i < numTail; i++ {
			idx := (start + i) % wc.tailM
			wc.processLine(wc.tailBuf[idx], minLength)
		}
		return nil
	}

	wc.mu.Lock()
	defer wc.mu.Unlock()

	wc.headCount++
	if wc.headCount <= wc.headN {
		// process synchronously
		wc.processLine(line, minLength)
	} else if wc.tailM > 0 {
		// Buffer in circular tail buffer
		wc.tailBuf[wc.tailPos] = line
		wc.tailPos = (wc.tailPos + 1) % wc.tailM
	}
	return nil
}

func (wc *WordCompleter) processLine(line string, minLength int) {
	parts, err := shellquote.Split(line)
	if err != nil {
		return
	}
	var words []string
	for _, word := range parts {
		if len(word) < minLength {
			continue
		}
		words = append(words, word)
	}
	if len(words) > 0 {
		wc.counter.AddWords(words)
	}
}

func (wc *WordCompleter) Show(word string, top int) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	for _, wf := range wc.counter.Suggest(word, top) {
		println(wf.Word, wf.Count)
	}
}
