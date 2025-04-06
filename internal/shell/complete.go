package shell

import (
	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

var filePathCompleter = completer.FilePathCompleter{
	IgnoreCase: true,
}

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

	completions := filePathCompleter.Complete(d)
	for i := range completions {
		completions[i].Description = "file"
	}
	completions = append(completions, prompt.FilterHasPrefix(unique(s), d.CurrentLine(), true)...)
	return completions
}
