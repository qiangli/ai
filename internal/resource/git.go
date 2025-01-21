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
Generate the commit message for those changes using the *Conventional Commits specification* provided below without additional explanations or code block fencing.

===
%s
`

func GetGitMessageSystem(short bool) string {
	if short {
		return gitMessageShort
	}
	return fmt.Sprintf(longFormat, gitMessageConventional)
}

//go:embed cli/git_system.md
var cliGitSystem string

func GetCliGitSystem() string {
	return cliGitSystem
}
