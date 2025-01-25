package llm

import (
	"context"
	"fmt"

	"github.com/openai/openai-go"

	"github.com/qiangli/ai/internal"
)

var gptrTools = []openai.ChatCompletionToolParam{
	define("gptr_generate_report",
		"Provides an easy way to conduct research on various topics and generate different types of reports.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"report_type": map[string]string{
					"type":        "string",
					"description": "The type of report to generate",
				},
				"tone": map[string]string{
					"type":        "string",
					"description": "The tone of the report",
				},
			},
			"required": []string{"report_type", "tone"},
		},
	),
}

func runGptrTool(cfg *internal.ToolConfig, ctx context.Context, name string, props map[string]interface{}) (string, error) {
	switch name {
	case "gptr_generate_report":
		return runGptr(cfg, ctx, props)
	default:
		return "", fmt.Errorf("unknown GPTR tool: %s", name)
	}
}

func runGptr(cfg *internal.ToolConfig, ctx context.Context, props map[string]interface{}) (string, error) {
	reportType, ok := props["report_type"].(string)
	if !ok {
		return "", fmt.Errorf("report_type is required")
	}
	tone, ok := props["tone"].(string)
	if !ok {
		return "", fmt.Errorf("tone is required")
	}
	return cfg.Next(ctx, fmt.Sprintf("/%s/%s", reportType, tone))
}

func GetGptrTools() []openai.ChatCompletionToolParam {
	return gptrTools
}
