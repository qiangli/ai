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
		log.Infof("%s\n", message.Content)
	}

	content := util.Render(message.Content)
	log.Infof("\n[%s]\n", message.Agent)
	log.Infoln(content)
}
