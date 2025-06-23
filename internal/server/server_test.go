package server

import (
	"testing"
)

func TestServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	StartServer("localhost:58080")
}

func TestSendMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	const wsUrl = "ws://localhost:58080/ws"

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
