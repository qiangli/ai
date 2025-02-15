package agent

// import (
// 	"context"
// 	"testing"

// 	"github.com/qiangli/ai/internal"
// 	"github.com/qiangli/ai/internal/log"
// )

// func TestResolveWorkspaceBase(t *testing.T) {
// 	if testing.Short() {
// 		t.Skip("skipping test in short mode.")
// 	}

// 	cfg := &internal.LLMConfig{
// 		L1ApiKey:  "sk-1234",
// 		L1Model:   "gpt-4o-mini",
// 		L1BaseUrl: "http://localhost:4000",
// 	}
// 	log.SetLogLevel(log.Verbose)

// 	// "is test_data empty?" - won't work
// 	// the following works:
// 	// "is ./test_data empty?"
// 	// "is test_data folder empty?"
// 	ws, err := resolveWorkspaceBase(context.TODO(), cfg, "", "is test_data folder empty?")
// 	if err != nil {
// 		t.Errorf("resolve error: %v", err)
// 		return
// 	}
// 	t.Logf("resolve ws: %s\n", ws)
// }
