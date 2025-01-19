package agent

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/log"
)

func TestScriptAgentSend(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &internal.LLMConfig{
		ApiKey:  "sk-1234",
		Model:   "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
	}

	agent, err := NewScriptAgent(&internal.AppConfig{
		LLM: cfg,
	})
	if err != nil {
		t.Errorf("NewScriptAgent error: %v", err)
		return
	}

	log.SetLogLevel(log.Verbose)

	command := ""
	message := "what is the latest node version?"
	resp, err := agent.Send(context.TODO(), &UserInput{
		Agent:      "test script",
		Subcommand: command,
		Message:    message,
	})
	if err != nil {
		t.Errorf("ScriptAgent.Send error: %v", err)
		return
	}

	t.Logf("Script Agent: %+v\n", resp)
}
