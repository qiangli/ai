package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	hubapi "github.com/qiangli/ai/internal/hub/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

type Message = hubapi.Message
type Payload = hubapi.Payload
type ContentPart = hubapi.ContentPart

func takeScreenshot(cfg *api.AppConfig) (string, error) {
	log.Debugf("⣿ taking screenshot: %s\n", cfg.HubAddress)

	imgFile := filepath.Join(cfg.Temp, "screenshot.png")

	screenshot := func() error {
		hubUrl, err := getHubUrl(cfg.HubAddress)
		if err != nil {
			return err
		}

		data, err := requestScreenshot(hubUrl)
		if err != nil {
			return err
		}

		const filePerm = 0644
		return os.WriteFile(imgFile, data, filePerm)
	}

	err := screenshot()

	if err != nil {
		log.Errorf("\033[31m✗\033[0m %s\n", err)
		return "", err
	}

	log.Infof("✔ %s \n", imgFile)
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

	//
	log.Infof("📸 Taking screenshot...\n")

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
					base64data := resp.Payload
					if commaIdx := strings.Index(base64data, ","); commaIdx != -1 {
						base64data = base64data[commaIdx+1:]
					}
					decoded, err := base64.StdEncoding.DecodeString(base64data)
					if err != nil {
						errorCh <- err
					} else {
						resultCh <- decoded
					}
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
