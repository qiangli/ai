package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

func voiceInput(cfg *api.AppConfig) (string, error) {
	log.Debugf("⣿ voice input: %s\n", cfg.HubAddress)

	hubUrl, err := getHubUrl(cfg.HubAddress)
	if err != nil {
		return "", err
	}

	data, err := requestVoiceInput(hubUrl)

	if err != nil {
		log.Errorf("\033[31m✗\033[0m %s\n", err)
		return "", err
	}

	log.Infof("✔ %s \n", string(data))

	return string(data), nil
}

// TODO merge/refactor requestScreenshot
func requestVoiceInput(wsUrl string) ([]byte, error) {
	u, err := url.Parse(wsUrl)
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var id = uuid.New().String()
	var sender = uuid.New().String()

	register := &Message{
		Type:    "register",
		Sender:  sender,
		Payload: "hi",
	}

	request := &Message{
		Type:      "request",
		ID:        id,
		Sender:    sender,
		Recipient: "chrome",
		Action:    "voice",
	}

	send := func(msg *Message) error {
		req, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		if err := conn.WriteMessage(websocket.TextMessage, req); err != nil {
			return err
		}
		return nil
	}

	if err := send(register); err != nil {
		return nil, err
	}

	if err := send(request); err != nil {
		return nil, err
	}

	// ready for speaking
	log.Infof("🎤 Please speak:\n")

	// wait for response
	const maxWait = 30
	resultCh := make(chan []byte)
	errorCh := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(maxWait)*time.Second)
	defer cancel()

	go func() {
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				errorCh <- err
				return
			}
			if msgType != websocket.TextMessage {
				continue
			}
			var resp Message
			if err := json.Unmarshal(data, &resp); err != nil {
				errorCh <- err
				return
			}
			if resp.Reference == id {
				if resp.Code == "200" {
					resultCh <- []byte(resp.Payload)
				} else {
					errorCh <- fmt.Errorf("failed: %s", resp.Payload)
				}
				return
			}
		}
	}()

	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errorCh:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout waiting for response")
	}
}
