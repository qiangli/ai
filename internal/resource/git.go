package resource

import (
	_ "embed"
	"fmt"
)

// https://github.com/Aider-AI/aider/blob/main/aider/prompts.py
//
//go:embed git/message_short.md
var gitMessageShort string

// https://www.conventionalcommits.org/en/v1.0.0/#summary
//
//go:embed git/message_conventional.md
var gitMessageConventional string

const longFormat = `
You are an expert software engineer that generates concise Git commit messages based on the provided diffs.

Review the diffs carefully.

Generate the commit message for those changes using the *Conventional Commits specification* provided below.

===
%s
===

The response must conform strictly to the provided specification without any additional explanations or code block fencing.
`

func getGitMessageSystem(short bool) string {
	if short {
		return gitMessageShort
	}
	return fmt.Sprintf(longFormat, gitMessageConventional)
}

//go:embed cli/git_sub_system.md
var cliGitSubSystem string

func GetCliGitSubSystem() string {
	return cliGitSubSystem
}

// GetGitMessageSystem returns the git message system prompt based on the subcommand.
// Returns the conventional system prompt unless "short" is requested.
func GetGitMessageSystem(sub string) (string, error) {
	switch sub {
	case "short":
		return getGitMessageSystem(true), nil
	case "conventional":
		return getGitMessageSystem(false), nil
	}
	return "", fmt.Errorf("unknown @git subcommand: %s", sub)
}
