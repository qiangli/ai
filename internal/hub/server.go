package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"

	"github.com/qiangli/ai/swarm/api"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("AI Hub"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func StartServer(cfg *api.AppConfig) {
	address := cfg.HubAddress
	settings := &Settings{
		HubUrl:   fmt.Sprintf("ws://%s/hub", address),
		HubState: 1,
	}

	hub := newHub(cfg)
	hub.settings = settings

	go hub.run()

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	})

	http.HandleFunc("/hub", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Printf("WebSocket server listening on %s/hub\n", address)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func SendMessage(wsURL string, message string) (string, error) {
	u, err := url.Parse(wsURL)
	if err != nil {
		return "", err
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		return "", err
	}

	_, data, err := conn.ReadMessage()
	if err != nil {
		return "", nil
	}

	return string(data), nil
}
