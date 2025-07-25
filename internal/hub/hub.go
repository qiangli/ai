package hub

import (
	"path/filepath"
	"time"

	ds "github.com/qiangli/ai/internal/db"
	hubapi "github.com/qiangli/ai/internal/hub/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

type Message = hubapi.Message
type Payload = hubapi.Payload
type ContentPart = hubapi.ContentPart

type ActionStatus string

const (
	StatusOK       ActionStatus = "200"
	StatusError    ActionStatus = "500"
	StatusNotFound ActionStatus = "404"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
// https://github.com/gorilla/websocket/blob/main/examples/chat/hub.go
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	message chan *Message

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	cfg      *api.AppConfig
	settings *Settings

	kvStore *ds.KVStore
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
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) Start() error {
	// start data services
	location := filepath.Join(h.cfg.Base, "store")
	kv := ds.NewKVStore(location)
	store := "kv.db"
	if err := kv.Open(store); err != nil {
		return err
	}
	defer kv.Close()

	log.Debugf("KV store %q ready for business at: %s\n", store, location)

	h.kvStore = kv

	go h.run()

	return nil
}

func (h *Hub) Stop() {
	if h.kvStore != nil {
		h.kvStore.Close()
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
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

// broadcastMessage sends msg to all clients including the sender
func (h *Hub) broadcastMessage(msg *Message) {
	log.Debugf("broadcastMessage %s\n", msg)

	for client := range h.clients {
		select {
		case client.msg <- msg:
		default:
			close(client.msg)
			delete(h.clients, client)
		}
	}
}

// sendPrivateMessage delivers msg to the specified recipient
func (h *Hub) sendPrivateMessage(msg *Message) {
	log.Debugf("sendPrivateMessage %s\n", msg)

	for client := range h.clients {
		if client.ID == msg.Recipient {
			select {
			case client.msg <- msg:
			default:
				close(client.msg)
				delete(h.clients, client)
			}
		}
	}
}

func (h *Hub) respond(req *Message) {
	log.Debugf("hub respond %s\n", req)

	for client := range h.clients {
		if client.ID == req.Sender {
			// process message
			now := time.Now()
			resp := &Message{
				Type:      "response",
				Sender:    "hub",
				Recipient: client.ID,
				Reference: req.ID,
				Timestamp: &now,
			}
			// TODO add more service (reserved name recipient)
			switch {
			case req.Recipient == "ai":
				code, result := h.handleAI(req.Payload)
				resp.Code = string(code)
				resp.Payload = result
			case req.Recipient == "kv":
				code, result := h.handleKV(req.Payload)
				resp.Code = string(code)
				resp.Payload = result
			}
			select {
			case client.msg <- resp:
			default:
				close(client.msg)
				delete(h.clients, client)
			}
		}
	}
}
