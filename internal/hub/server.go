package hub

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	hubapi "github.com/qiangli/ai/internal/hub/api"
	hubws "github.com/qiangli/ai/internal/hub/ws"
	"github.com/qiangli/ai/swarm/api"
)

func StartServer(cfg *api.AppConfig) {
	address := cfg.HubAddress
	hubUrl := fmt.Sprintf("ws://%s/hub", address)
	settings := &Settings{
		HubUrl:   hubUrl,
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
	http.HandleFunc("/message", createMessageHandler(hubUrl))

	http.HandleFunc("/hub", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Printf("Server listening on %s\n", address)
	log.Printf("Websocket: %s\n", hubUrl)
	err := http.ListenAndServe(address, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func SendMessage(wsURL string, message string) (string, error) {
	var req hubapi.Message
	if err := json.Unmarshal([]byte(message), &req); err != nil {
		return "", err
	}
	resp, err := hubws.SendMessage(wsURL, "", &req)
	if err != nil {
		return "", err
	}
	return resp.Payload, nil
}
