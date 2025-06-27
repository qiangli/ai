package hub

import (
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	cfg := &api.AppConfig{
		HubAddress: "localhost:58080",
	}
	StartServer(cfg)
}

func TestSendMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	const wsUrl = "ws://localhost:58080/hub"

	tests := []struct {
		messages []string
	}{
		{
			[]string{
				`{"type": "register", "sender": "chrome", "payload": "hello"}`,
				`{"type": "private", "sender": "chrome", "recipient": "chrome", "id": "1000", "payload": "@swe write code in python"}`,
			},
		},
	}

	for _, tc := range tests {
		for _, message := range tc.messages {
			resp, err := SendMessage(wsUrl, message)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("received: %s\n", resp)
		}
	}
}
