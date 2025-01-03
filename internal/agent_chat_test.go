package internal

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal/log"
)

func TestChatAgentSend(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &Config{
		ApiKey:  "sk-1234",
		Model:   "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
	}
	chat, err := NewChatAgent(cfg, "", "")
	if err != nil {
		t.Errorf("New chat agent error: %v", err)
		return
	}

	log.SetLogLevel(log.Verbose)

	input := "what is this command for zic"
	resp, err := chat.Send(context.TODO(), input)
	if err != nil {
		t.Errorf("chat agent send error: %v", err)
		return
	}

	t.Logf("chat agent: %+v\n", resp)
}
