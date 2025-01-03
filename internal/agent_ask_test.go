package internal

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal/log"
)

func TestAskAgentSend(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &Config{
		ApiKey:  "sk-1234",
		Model:   "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
	}
	agent, err := NewAskAgent(cfg, "", "")
	if err != nil {
		t.Errorf("New AskAgent error: %v", err)
		return
	}

	log.SetLogLevel(log.Verbose)

	input := "what is zic command?"
	resp, err := agent.Send(context.TODO(), input)
	if err != nil {
		t.Errorf("Ask agent send error: %v", err)
		return
	}

	t.Logf("Ask agent: %+v\n", resp)
}
