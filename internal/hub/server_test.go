package hub

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/qiangli/ai/swarm/api"
)

func TestServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	hubConfig := &api.HubConfig{
		Enable:  true,
		Address: "localhost:58080",
	}
	cfg := &api.AppConfig{
		Hub: hubConfig,
	}
	StartServer(cfg)
}

func TestSendMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tests := []struct {
		messages []string
	}{
		{
			[]string{
				`{"type": "register", "sender": "chrome-test", "payload": "hello"}`,
				`{"type": "private", "sender": "chrome-test", "recipient": "chrome-test", "id": "1000", "reference": "1000", "code": "200", "payload": "echo"}`,
			},
		},
	}

	const wsUrl = "ws://localhost:58080/hub"
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

func TestSendMessageByCurl(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tests := []struct {
		messages []string
	}{
		{
			[]string{
				`{"type": "register", "sender": "chrome-test", "payload": "hello"}`,
				`{"type": "private", "sender": "chrome-test", "recipient": "chrome-test", "id": "1000", "reference": "1000", "code": "200", "payload": "echo"}`,
				`{"type": "request", "sender": "chrome-test", "recipient": "chrome", "id": "1001", "action": "screenshot"}`,
				`{"type": "request", "sender": "chrome-test", "recipient": "chrome", "id": "1002", "action": "get-selection"}`,
				// `{"type": "request", "sender": "chrome-test", "recipient": "chrome", "id": "1002", "action": "voice"}`,
			},
		},
	}

	const httpUrl = "http://localhost:58080/message"
	for _, tc := range tests {
		for _, message := range tc.messages {
			cmd := exec.Command("curl", "-s", "-X", "POST", "-H", "Content-Type: application/json", "-d", message, httpUrl)
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err)
			}
			var resp Message
			if err := json.Unmarshal([]byte(out), &resp); err != nil {
				t.Fatal(err)
			}
			t.Logf("received: %s\n", resp)
		}
	}
}
