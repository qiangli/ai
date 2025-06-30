package agent

import (
	hubws "github.com/qiangli/ai/internal/hub/ws"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

func voiceInput(cfg *api.AppConfig) (string, error) {
	log.Debugf("⣿ voice input: %s\n", cfg.HubAddress)

	wsUrl, err := hubws.GetHubUrl(cfg.HubAddress)
	if err != nil {
		return "", err
	}
	prompt := "🎤 Please speak:\n"

	data, err := hubws.VoiceInput(wsUrl, prompt)

	if err != nil {
		log.Errorf("\033[31m✗\033[0m %s\n", err)
		return "", err
	}

	log.Infof("✔ %s \n", string(data))

	return string(data), nil
}
