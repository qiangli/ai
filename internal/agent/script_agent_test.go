package agent

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
)

func TestScriptAgentSend(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &llm.Config{
		ApiKey:  "sk-1234",
		Model:   "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
	}

	agent, err := NewScriptAgent(cfg, "", "")
	if err != nil {
		t.Errorf("NewScriptAgent error: %v", err)
		return
	}

	log.SetLogLevel(log.Verbose)

	command := ""
	message := "what is the latest node version?"
	resp, err := agent.Send(context.TODO(), &UserInput{
		Command: command,
		Message: message,
	})
	if err != nil {
		t.Errorf("ScriptAgent.Send error: %v", err)
		return
	}

	t.Logf("ScriptAgent: %+v\n", resp)
}
