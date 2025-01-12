package agent

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/tool"
)

func TestCheckWworkspace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &llm.Config{
		Model:   "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
		ApiKey:  "sk-1234",

		Debug:  true,
		DryRun: false,

		Tools: tool.SystemTools,

		Workspace: "test_data",
	}

	log.SetLogLevel(log.Verbose)

	input := "add a new 'agent' command in test_data/cmd"
	resp, err := checkWorkspace(context.TODO(), cfg, input, llm.L1)
	if err != nil {
		t.Errorf("check agent send error: %v", err)
		return
	}

	t.Logf("check agent: %+v\n", resp)
}
