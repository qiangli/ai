package agent

// import (
// 	"os"
// 	"path/filepath"

// 	// hubws "github.com/qiangli/ai/internal/hub/ws"
// 	"github.com/qiangli/ai/swarm/log"
// 	"github.com/qiangli/ai/swarm/api"
// )

// func takeScreenshot(cfg *api.AppConfig) (string, error) {
// 	log.Debugf("⣿ taking screenshot: %s\n", cfg.Hub.Address)

// 	imgFile := filepath.Join(cfg.Temp, "screenshot.png")

// 	screenshot := func() error {
// 		wsUrl, err := hubws.GetHubUrl(cfg.Hub.Address)
// 		if err != nil {
// 			return err
// 		}
// 		prompt := "📸 Taking screenshot...\n"

// 		result, err := hubws.Screenshot(wsUrl, prompt)
// 		if err != nil {
// 			return err
// 		}
// 		data, err := hubws.DecodeImage(result)
// 		if err != nil {
// 			return err
// 		}
// 		const filePerm = 0644
// 		return os.WriteFile(imgFile, data, filePerm)
// 	}

// 	err := screenshot()

// 	if err != nil {
// 		log.Errorf("\033[31m✗\033[0m %s\n", err)
// 		return "", err
// 	}

// 	log.Infof("✔ %s \n", imgFile)
// 	return imgFile, nil
// }
