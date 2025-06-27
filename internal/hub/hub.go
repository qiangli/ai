package hub

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/qiangli/ai/agent"
	hubapi "github.com/qiangli/ai/internal/hub/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

type Message = hubapi.Message
type Payload = hubapi.Payload
type ContentPart = hubapi.ContentPart

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
// https://github.com/gorilla/websocket/blob/main/examples/chat/hub.go
type Hub struct {
	// Registered clients.
	clients map[string]*Client

	// Inbound messages from the clients.
	message chan *Message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	cfg      *api.AppConfig
	settings *Settings
}

type Settings struct {
	HubUrl string `json:"hubUrl"`

	// 0 stopped 1 running
	HubState int `json:"hubState"`
}

func newHub(cfg *api.AppConfig) *Hub {
	return &Hub{
		cfg:        cfg,
		message:    make(chan *Message),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[string]*Client),
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client.ID] = client
			msg := &Message{
				Type:      "response",
				Sender:    "hub",
				Recipient: client.ID,
				Payload:   "200 registration successful",
				Timestamp: time.Now(),
			}
			h.sendPrivateMessage(msg)
		case client := <-h.unregister:
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.msg)
			}
		case msg := <-h.message:
			switch msg.Type {
			case "broadcast":
				h.broadcastMessage(msg)
			case "private", "request", "response":
				h.sendPrivateMessage(msg)
			case "hub":
				h.respond(msg)
			default:
				log.Errorf("Unknown msg type: %s", msg.Type)
			}
		}
	}
}

// broadcastMessage sends msg to all clients except the sender
func (h *Hub) broadcastMessage(msg *Message) {
	log.Debugf("broadcastMessage %s\n", msg)

	sender := msg.Sender
	for id, client := range h.clients {
		// skip self
		if id == sender {
			continue
		}
		select {
		case client.msg <- msg:
		default:
			close(client.msg)
			delete(h.clients, client.ID)
		}
	}
}

// sendPrivateMessage delivers msg to the specified recipient
func (h *Hub) sendPrivateMessage(msg *Message) {
	log.Debugf("sendPrivateMessage %s\n", msg)

	if client, ok := h.clients[msg.Recipient]; ok {
		select {
		case client.msg <- msg:
		default:
			close(client.msg)
			delete(h.clients, client.ID)
		}
	}
}

func (h *Hub) respond(req *Message) {
	log.Debugf("hub respond %s\n", req)

	if client, ok := h.clients[req.Sender]; ok {
		// process message
		resp := &Message{
			Type:      "response",
			Sender:    "hub",
			Recipient: client.ID,
			Timestamp: time.Now(),
		}
		if req.Recipient == "ai" {
			resp.Payload = h.ai(req.Payload)
		}
		select {
		case client.msg <- resp:
		default:
			close(client.msg)
			delete(h.clients, client.ID)
		}
	}
}

func (h *Hub) ai(data string) string {
	cfg := h.cfg

	in := &api.UserInput{
		Agent:   cfg.Agent,
		Command: cfg.Command,
	}

	var payload Payload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return err.Error()
	}

	in.Content = payload.Content

	var messages []*api.Message
	for _, v := range payload.Parts {
		// TODO optimize and skip this step
		// LLM use the same encoding for multi media data
		// data:[<media-type>][;base64],<data>
		ca := strings.SplitN(v.Content, ",", 2)
		if len(ca) != 2 {
			return "invalid multi media content"
		}
		raw, err := base64.StdEncoding.DecodeString(ca[1])
		if err != nil {
			return err.Error()
		}
		messages = append(messages, &api.Message{
			ContentType: v.ContentType,
			Content:     string(raw),
			Role:        api.RoleUser,
		})
	}
	in.Messages = messages

	// run
	cfg.Format = "text"
	if err := agent.RunSwarm(cfg, in); err != nil {
		log.Errorf("Error running agent: %s\n", err)
		return err.Error()
	}

	//success
	log.Infof("ai executed successfully\n")

	return cfg.Stdout
}
