package hub

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/qiangli/ai/internal/db"
	hubapi "github.com/qiangli/ai/internal/hub/api"
	hubws "github.com/qiangli/ai/internal/hub/ws"
	"github.com/qiangli/ai/internal/hub/ws/bridge"
	"github.com/qiangli/ai/internal/log"
	llmproxy "github.com/qiangli/ai/internal/proxy/llm"
	"github.com/qiangli/ai/internal/xterm"
	"github.com/qiangli/ai/swarm/api"
)

func StartServer(cfg *api.AppConfig) error {
	// add signal handler

	if !cfg.Hub.Enable {
		log.Infof("Hub service is disabled")
		return nil
	}
	address := cfg.Hub.Address
	hubUrl := fmt.Sprintf("ws://%s/hub", address)
	settings := &Settings{
		HubUrl:   hubUrl,
		HubState: 1,
	}

	hub := newHub(cfg)
	hub.settings = settings

	// start websocket service
	if err := hub.Start(); err != nil {
		log.Errorf("Hub service: %v ", err)
		return err
	}
	defer hub.Stop()

	//
	if cfg.Hub.Pg {
		go db.StartPG(cfg.Hub.PgAddress, "pglite")
	}
	if cfg.Hub.Mysql {
		go db.StartMySQL(cfg.Hub.MysqlAddress, "mydb")
	}
	if cfg.Hub.Redis {
		go db.StartRedis(cfg.Hub.RedisAddress)
	}
	if cfg.Hub.Pg || cfg.Hub.Mysql {
		go bridge.Start()
	}
	if cfg.Hub.Terminal {
		go xterm.Start(cfg)
	}
	if cfg.Hub.LLMProxy {
		go llmproxy.Start(cfg)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(settings)
	})
	mux.HandleFunc("/message", createMessageHandler(hubUrl))

	mux.HandleFunc("/hub", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	log.Infof("Hub Server listening on %s\n", address)
	log.Infof("Websocket: %s\n", hubUrl)

	err := http.ListenAndServe(address, mux)
	if err != nil {
		return err
	}
	return nil
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
