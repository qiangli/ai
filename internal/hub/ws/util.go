package ws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	hubapi "github.com/qiangli/ai/internal/hub/api"
	"github.com/qiangli/ai/internal/log"
)

type Message = hubapi.Message
type Payload = hubapi.Payload
type ContentPart = hubapi.ContentPart

func GetHubUrl(address string) (string, error) {
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

func DecodeImage(base64data string) ([]byte, error) {
	if commaIdx := strings.Index(base64data, ","); commaIdx != -1 {
		base64data = base64data[commaIdx+1:]
	}
	return base64.StdEncoding.DecodeString(base64data)
}

// Send Recipient/Action/Payload
func SendRequest(wsUrl string, prompt string, request *Message) (*Message, error) {
	var id = uuid.New().String()
	var sender = uuid.New().String()

	message := &Message{
		Type:   "request",
		ID:     id,
		Sender: sender,
		//
		Recipient: request.Recipient,
		Action:    request.Action,
		Payload:   request.Payload,
	}

	resp, err := SendMessage(wsUrl, prompt, message)
	if err != nil {
		return nil, err
	}
	if resp.Code == "200" {
		return resp, nil
	}
	return nil, fmt.Errorf("failed: %s %s", resp.Code, resp.Payload)
}

func SendMessage(wsUrl string, prompt string, message *Message) (*Message, error) {
	u, err := url.Parse(wsUrl)
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	send := func(msg *Message) error {
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return err
		}
		return nil
	}

	// ID is required for a response
	if message.ID == "" {
		message.ID = uuid.New().String()
	}

	if err := send(message); err != nil {
		return nil, err
	}

	// TODO a more robust solution is to check a ready confirmation
	// from the recipient e.g. for voice input
	if prompt != "" {
		log.Info(prompt)
	}

	// wait for response
	const maxWait = 30
	resultCh := make(chan *Message)
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

			log.Debugf("send memsage msgType %v %s", msgType, string(data))

			var resp Message
			if err := json.Unmarshal(data, &resp); err != nil {
				errorCh <- err
				return
			}
			if resp.Reference == message.ID {
				resultCh <- &resp
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
