package server

import (
	"time"

	"github.com/qiangli/ai/internal/log"
)

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
}

func newHub() *Hub {
	return &Hub{
		message:  make(chan *Message),
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
				Type: "response",
				Sender: "hub",
				Recipient: client.ID,
				Payload: "200 registration successful",
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
			case "private":
				h.sendPrivateMessage(msg)
			default:
				log.Errorf("Unknown msg type: %s", msg.Type)
			}
		}
	}
}

// broadcastMessage sends msg to all clients except the sender
func (h *Hub) broadcastMessage(msg *Message) {
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
	if client, ok := h.clients[msg.Recipient]; ok {
		select {
		case client.msg <- msg:
		default:
			close(client.msg)
			delete(h.clients, client.ID)
		}
	}
}
