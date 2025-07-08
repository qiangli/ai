package bridge

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"github.com/qiangli/ai/internal/log"
)

type Message struct {
	ID     int    `json:"id"`
	Action string `json:"action"`

	// open
	DB          string       `json:"db"`
	Credentials *Credentials `json:"credentials"`

	// exec
	SQL string `json:"sql"`

	// close
	// get_postgres_credentials
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins
	},
}

func handleConnection(w http.ResponseWriter, r *http.Request, ctx *sync.Map) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Debugf("Upgrade: %v\n", err)
		return
	}
	defer conn.Close()

	log.Debugln("Opened connection")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Debugf("ReadMessage: %v\n", err)
			break
		}

		log.Debugf("received: %s", message)

		var req Message
		if err := json.Unmarshal(message, &req); err != nil {
			log.Debugf("unmarshal: %v\n", err)
			continue
		}

		result, err := response(&req, ctx)
		if err != nil {
			log.Debugf("response error: %v", err)
			result = map[string]any{"id": req.ID, "error": err.Error()}
		} else {
			result["id"] = req.ID
		}

		resp, err := json.Marshal(result)
		if err != nil {
			log.Debugf("marshal: %v\n", err)
			continue
		}

		if err := conn.WriteMessage(websocket.TextMessage, resp); err != nil {
			log.Debugf("send error: %v\n", err)
			break
		}

		log.Debugf("sent: %s\n", string(resp))
	}
}

func Start() {
	addr := fmt.Sprintf(":%d", 14645)

	var localCtx = &sync.Map{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleConnection(w, r, localCtx)
	})

	log.Infof("Websocket bridge listening on%s...\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Errorf("failed to start: %v", err)
	}
}
