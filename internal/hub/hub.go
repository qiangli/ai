package hub

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/qiangli/ai/agent"
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
				code, result := h.doAI(req.Payload)
				resp.Code = string(code)
				resp.Payload = result
			case req.Recipient == "kv":
				code, result := h.doKV(req.Payload)
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

func (h *Hub) doAI(data string) (ActionStatus, string) {
	log.Debugf("hub doAI: %v", data)

	cfg := h.cfg

	in := &api.UserInput{
		Agent:   cfg.Agent,
		Command: cfg.Command,
	}

	var payload Payload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return StatusError, err.Error()
	}

	in.Content = payload.Content

	var messages []*api.Message
	for _, v := range payload.Parts {
		// TODO optimize and skip this step
		// LLM use the same encoding for multi media data
		// data:[<media-type>][;base64],<data>
		ca := strings.SplitN(v.Content, ",", 2)
		if len(ca) != 2 {
			return StatusError, "invalid multi media content"
		}
		raw, err := base64.StdEncoding.DecodeString(ca[1])
		if err != nil {
			return StatusError, err.Error()
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
		log.Errorf("error running agent: %s\n", err)
		return StatusError, err.Error()
	}

	//success
	log.Infof("ai executed successfully\n")

	return StatusOK, cfg.Stdout
}

type KVPayload struct {
	Bucket string `json:"bucket"`

	Key   string `json:"key"`
	Value string `json:"value"`

	// bucket: create/drop key-value: set/get/remove
	Action string `json:"action"`
}

func (h *Hub) doKV(data string) (ActionStatus, string) {
	log.Debugf("hub doAI: %v", data)

	var payload KVPayload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return StatusError, err.Error()
	}

	//
	var bucket = payload.Bucket
	var key = payload.Key
	var val = payload.Value

	if bucket == "" {
		return StatusError, "missing bucket name"
	}
	switch payload.Action {
	case "create":
		if err := h.kvStore.Create(bucket); err != nil {
			return StatusError, err.Error()
		}
		return StatusOK, fmt.Sprintf("bucket %q created", bucket)
	case "drop":
		if err := h.kvStore.Drop(bucket); err != nil {
			return StatusError, err.Error()
		}
		return StatusOK, fmt.Sprintf("bucket %q removed", bucket)
	}

	if key == "" {
		return StatusError, "missing key"
	}
	switch payload.Action {
	case "set":
		err := h.kvStore.Put(bucket, key, val)
		if err != nil {
			return StatusError, err.Error()
		}
		return StatusOK, "Done"
	case "get":
		val, err := h.kvStore.Get(bucket, key)
		if err != nil {
			return StatusError, err.Error()
		}
		if val == nil {
			return StatusNotFound, fmt.Sprintf("Not found: %s in %s", key, bucket)
		}
		return StatusOK, *val
	case "remove":
		err := h.kvStore.Delete(bucket, key)
		if err != nil {
			return StatusError, err.Error()
		}
		return StatusOK, "Done"
	}

	return StatusError, fmt.Sprintf("Action not supported: %s", payload.Action)
}
