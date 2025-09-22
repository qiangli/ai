package agent

// import (
// 	// hubws "github.com/qiangli/ai/internal/hub/ws"
// 	"github.com/qiangli/ai/swarm/log"
// 	"github.com/qiangli/ai/swarm/api"
// )

// func voiceInput(cfg *api.AppConfig) (string, error) {
// 	log.GetLogger(ctx).Debug("⣿ voice input: %s\n", cfg.Hub.Address)

// 	wsUrl, err := hubws.GetHubUrl(cfg.Hub.Address)
// 	if err != nil {
// 		return "", err
// 	}
// 	prompt := "🎤 Please speak:\n"

// 	data, err := hubws.VoiceInput(wsUrl, prompt)

// 	if err != nil {
// 		log.GetLogger(ctx).Error("\033[31m✗\033[0m %s\n", err)
// 		return "", err
// 	}

// 	log.GetLogger(ctx).Info("✔ %s \n", string(data))

// 	return string(data), nil
// }
