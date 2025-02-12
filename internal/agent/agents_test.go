package agent

import (
	_ "embed"
	"testing"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
)

//go:embed resource/agent.yaml
var agentsYaml []byte

func TestLoadAgentsConfig(t *testing.T) {
	data := [][]byte{agentsYaml}
	cfg, err := LoadAgentsConfig(data)
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

	err := RunAskAgent(&internal.AppConfig{
		LLM: cfg,
	}, "ask", input)

	if err != nil {
		t.Errorf("Ask agent send error: %v", err)
		return
	}
}

func TestRunGitAgent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &internal.LLMConfig{
		ApiKey:  "sk-1234",
		Model:   "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
	}

	log.SetLogLevel(log.Verbose)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"Test short", "Quick commit: Updated file paths.", "short"},
		// {"Test conventional", "I'd like to make a commit with type 'fix' to address the bug in the login function.", "conventional"},
	}

	for _, v := range tests {
		t.Run(v.name, func(t *testing.T) {
			input := &UserInput{
				Message: v.input,
			}

			err := RunGitAgent(&internal.AppConfig{
				LLM: cfg,
			}, "git", input)

			if err != nil {
				t.Errorf("Ask agent send error: %v", err)
				return
			}
		})
	}
}
