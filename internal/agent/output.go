package agent

import (
	"os"
	"path/filepath"

	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

func renderMessage(message *api.Response) {
	if message == nil {
		return
	}

	// show original message if in verbose mode
	if log.IsVerbose() {
		log.Infof("\n[%s]\n", message.Agent)
		log.Infoln(message.Content)
	}

	// TODO: markdown formatting lost if the content is also tee'd to a file
	content := util.Render(message.Content)
	log.Infof("\n[%s]\n", message.Agent)
	log.Infoln(content)
}

func showMessage(message *api.Response) {
	if message == nil {
		return
	}
	log.Infof("\n[%s]\n", message.Agent)
	log.Infoln(message.Content)
}

func PrintMessage(output string, message *api.Response) {
	if output == "markdown" {
		renderMessage(message)
	} else {
		showMessage(message)
	}
}

func SaveMessage(filename string, message *api.Response) error {
	if message == nil {
		return nil
	}
	if filename == "" {
		return nil
	}
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(message.Content), os.ModePerm)
}
