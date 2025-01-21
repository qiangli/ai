package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"

	"github.com/qiangli/ai/internal"
)

var prTools = []openai.ChatCompletionToolParam{
	define("pr_describe",
		"Generate PR description - title, type, summary, code walkthrough and labels",
		nil,
	),
	define("pr_review",
		"Feedback about the PR, possible issues, security concerns, review effort and more",
		nil,
	),
	define("pr_improve",
		"Code suggestions for improving the PR",
		nil,
	),
	define("pr_changelog",
		"Update the CHANGELOG.md file with the PR changes",
		nil,
	),
}

func runPrTool(cfg *internal.ToolConfig, ctx context.Context, name string, props map[string]interface{}) (string, error) {
	switch name {
	case "pr_describe":
		return runPr(cfg, ctx, name)
	case "pr_review":
		return runPr(cfg, ctx, name)
	case "pr_improve":
		return runPr(cfg, ctx, name)
	case "pr_changelog":
		return runPr(cfg, ctx, name)
	default:
		return "", fmt.Errorf("unknown PR tool: %s", name)
	}
}

func runPr(cfg *internal.ToolConfig, ctx context.Context, name string) (string, error) {
	subcommand := name[3:]
	return cfg.Next(ctx, subcommand)
}

func GetPrTools() []openai.ChatCompletionToolParam {
	return prTools
}
