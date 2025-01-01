package internal

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal/log"
)

func TestScriptAgentSend(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &Config{
		ApiKey:  "sk-1234",
		Model:   "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
	}

	chat, err := NewScriptAgent(cfg, "", "")
	if err != nil {
		t.Errorf("NewScriptAgent error: %v", err)
		return
	}

	log.SetLogLevel(log.Verbose)

	command := "zic"
	message := "what is this command for"
	resp, err := chat.Send(context.TODO(), command, message)
	if err != nil {
		t.Errorf("ScriptAgent.Send error: %v", err)
		return
	}

	t.Logf("ScriptAgent: %+v\n", resp)
}
