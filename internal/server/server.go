package server

import (
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

func StartServer(address string) {
	hub := newHub()
	go hub.run()

	// http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Printf("WebSocket server listening on %s/ws\n", address)
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
