package agent

import (
	_ "embed"
	"testing"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
)

func TestLoadAgentsConfig(t *testing.T) {
	cfg, err := LoadAgentsConfig()
	if err != nil {
		t.Errorf("Error loading agents configuration: %v", err)
	}
	t.Logf("Agents configuration: %+v", cfg)
}

func TestRunAskAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &internal.LLMConfig{
		ApiKey:  "sk-1234",
		Model:   "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
	}

	log.SetLogLevel(log.Verbose)
	// internal.DryRun = true
	// internal.DryRunContent = "fake it"

	input := &UserInput{
		Message: "what is zic command?",
	}

	err := Run(&internal.AppConfig{
		LLM: cfg,
	}, "ask", input)

	if err != nil {
		t.Errorf("Ask agent send error: %v", err)
		return
	}
}
