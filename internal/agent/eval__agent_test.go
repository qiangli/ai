package agent

// import (
// 	"context"
// 	"testing"

// 	"github.com/qiangli/ai/internal"
// 	"github.com/qiangli/ai/internal/log"
// )

// func TestEvalAgentSend(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}

// 	cfg := &internal.LLMConfig{
// 		ApiKey:  "sk-1234",
// 		Model:   "gpt-4o-mini",
// 		BaseUrl: "http://localhost:4000",
// 	}
// 	chat, err := NewEvalAgent(&internal.AppConfig{
// 		LLM: cfg,
// 	})
// 	if err != nil {
// 		t.Errorf("New chat agent error: %v", err)
// 		return
// 	}

// 	log.SetLogLevel(log.Verbose)

// 	input := &UserInput{Message: "what is this ZIC command for"}
// 	resp, err := chat.Send(context.TODO(), input)
// 	if err != nil {
// 		t.Errorf("chat agent send error: %v", err)
// 		return
// 	}

// 	t.Logf("chat agent: %+v\n", resp)
// }
