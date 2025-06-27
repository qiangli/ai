package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	hubapi "github.com/qiangli/ai/internal/hub/api"
	"github.com/qiangli/ai/swarm/api"
)

type Message = hubapi.Message
type Payload = hubapi.Payload
type ContentPart = hubapi.ContentPart

func takeScreenshot(cfg *api.AppConfig) (string, error) {
	hubUrl, err := getHubUrl(cfg.HubAddress)
	if err != nil {
		return "", err
	}

	const filePerm = 0644
	data, err := requestScreenshot(hubUrl)
	if err != nil {
		return "", err
	}
	imgFile := filepath.Join(cfg.Temp, "screenshot.png")

	err = os.WriteFile(imgFile, data, filePerm)
	if err != nil {
		return "", err
	}

	return imgFile, nil
}

func getHubUrl(address string) (string, error) {
	type Settings struct {
		HubUrl string `json:"hubUrl"`
	}

	settingsUrl := fmt.Sprintf("http://%s/settings", address)
	resp, err := http.Get(settingsUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var settings Settings
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		return "", err
	}

	return settings.HubUrl, nil
}

func requestScreenshot(wsUrl string) ([]byte, error) {
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

	screenshot := &Message{
		Type:      "request",
		ID:        id,
		Sender:    sender,
		Recipient: "chrome",
		Action:    "screenshot",
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
	if err := send(screenshot); err != nil {
		return nil, err
	}

	// wait for response
	resultCh := make(chan []byte)
	errorCh := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	defer close(resultCh)
	defer close(errorCh)

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
					errorCh <- fmt.Errorf("failed to take screenshot %s", resp.Payload)
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
		return nil, fmt.Errorf("timeout waiting for screenshot response")
	}
}
