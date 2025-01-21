package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"

	"github.com/qiangli/ai/internal"
)

var gitTools = []openai.ChatCompletionToolParam{
	define("git_short",
		"Generate a short commit message for Git based on the provided information",
		nil,
	),
	define("git_conventional",
		"Generate a commit message for Git based on the provided information according to the Conventional Commits specification",
		nil,
	),
}

func runGitTool(cfg *internal.ToolConfig, ctx context.Context, name string, props map[string]interface{}) (string, error) {
	switch name {
	case "git_short":
		return runGit(cfg, ctx, name)
	case "git_conventional":
		return runGit(cfg, ctx, name)
	default:
		return "", fmt.Errorf("unknown GIT tool: %s", name)
	}
}

func runGit(cfg *internal.ToolConfig, ctx context.Context, name string) (string, error) {
	subcommand := name[4:]
	return cfg.Next(ctx, subcommand)
}

func GetGitTools() []openai.ChatCompletionToolParam {
	return gitTools
}
