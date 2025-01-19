package agent

import (
	"context"
	"testing"

	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
)

func TestEvalAgentSend(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &llm.Config{
		ApiKey:  "sk-1234",
		Model:   "gpt-4o-mini",
		BaseUrl: "http://localhost:4000",
	}
	chat, err := NewEvalAgent(cfg, "", "")
	if err != nil {
		t.Errorf("New chat agent error: %v", err)
		return
	}

	log.SetLogLevel(log.Verbose)

	input := &UserInput{Message: "what is this ZIC command for"}
	resp, err := chat.Send(context.TODO(), input)
	if err != nil {
		t.Errorf("chat agent send error: %v", err)
		return
	}

	t.Logf("chat agent: %+v\n", resp)
}
