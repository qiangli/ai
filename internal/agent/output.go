package agent

import (
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/util"
)

func renderMessage(message *ChatMessage) {
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

func showMessage(message *ChatMessage) {
	if message == nil {
		return
	}
	log.Infof("\n[%s]\n", message.Agent)
	log.Infoln(message.Content)
}

func PrintMessage(output string, message *ChatMessage) {
	if output == "markdown" {
		renderMessage(message)
	} else {
		showMessage(message)
	}
}
