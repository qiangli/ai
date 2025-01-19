package resource

import (
	_ "embed"
	"fmt"
)

// https://github.com/Aider-AI/aider/blob/main/aider/prompts.py
//
//go:embed git/commit_short.md
var gitCommitShort string

func getGitCommitShortSystemRole() string {
	return gitCommitShort
}

// https://www.conventionalcommits.org/en/v1.0.0/#summary
//
//go:embed git/commit_conventional.md
var gitCommitConventional string

func getConventionalCommitSystemRole() string {
	const format = `
You are an expert software engineer that generates concise Git commit messages based on the provided diffs.

Review the diffs carefully.
Generate the commit message for those changes using the *Conventional Commits specification* provided below without additional explanations or code block fencing.

===
%s
`
	return fmt.Sprintf(format, gitCommitConventional)
}

func GetGitSystemRoleContent(short bool) string {
	if short {
		return getGitCommitShortSystemRole()
	}
	return getConventionalCommitSystemRole()
}
