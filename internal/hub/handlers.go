package hub

import (
	"encoding/json"
	"net/http"
	"os"

	hubws "github.com/qiangli/ai/internal/hub/ws"
	"github.com/qiangli/ai/internal/log"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("AI Hub"))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func shutdownHandler(w http.ResponseWriter, r *http.Request) {
	log.Infof("shutdown requested. %+v", r.Header)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	os.Exit(0)
}

func createMessageHandler(wsUrl string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		prompt := r.URL.Query().Get("prompt")

		var msg Message
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		respMsg, err := hubws.SendMessage(wsUrl, prompt, &msg)
		if err != nil {
			http.Error(w, "Failed to send message: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(respMsg)
	}
}
