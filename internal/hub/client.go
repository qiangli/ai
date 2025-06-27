package hub

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/qiangli/ai/internal/log"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024 * 1024 * 20
)

// upgrader upgrades HTTP connections to WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	ID string

	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	msg chan *Message
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		if c.ID != "" {
			c.hub.unregister <- c
		}
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	genId := func() string {
		return uuid.New().String()
	}

	register := func(sender string) {
		if sender == "" {
			sender = genId()
		}
		c.ID = sender
		c.hub.register <- c
	}

	for {
		msgType, data, err := c.conn.ReadMessage()
		log.Debugf("readPump msgType: %v err: %v\n", msgType, err)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Errorf("error: %v", err)
			}
			break
		}

		if msgType != websocket.TextMessage {
			log.Debugf("message type: %v\n", msgType)
			continue
		}

		var msg Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Errorf("unmarshal: %v\n", err)
			continue
		}

		//
		if msg.Type == "heartbeat" {
			log.Debugf("heartbeat %s\n", c.ID)
			continue
		}

		if msg.Type == "" {
			msg.Type = "broadcast"
		}

		msg.Timestamp = time.Now()

		// required
		if msg.Type == "register" {
			if c.ID != "" {
				// ignore
				log.Debugf("already registered\n")
				continue
			}
			register(msg.Sender)
			continue
		}

		// optional
		if msg.Type == "unregister" {
			log.Debugf("unregister received\n")
			break
		}

		if c.ID == "" {
			// auto register
			register(msg.Sender)
		}

		c.hub.message <- &msg
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.msg:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			data, err := json.Marshal(msg)
			if err != nil {
				log.Errorf("marshal %v\n", err)
				continue
			}

			w.Write(data)

			// Add queued chat messages to the current websocket message.
			n := len(c.msg)
			for i := 0; i < n; i++ {
				if msg, ok := <-c.msg; ok {
					data, err := json.Marshal(msg)
					if err != nil {
						log.Errorf("marshal %v\n", err)
						continue
					}
					w.Write(data)
				}
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	log.Debugf("serveWs remoteAddr: %v\n", r.RemoteAddr)
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, msg: make(chan *Message, 256)}

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()
}
