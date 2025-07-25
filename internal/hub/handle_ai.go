package hub

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/qiangli/ai/internal/agent"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

func (h *Hub) doAI(data string) (ActionStatus, string) {
	log.Debugf("hub doAI: %v", data)

	cfg := h.cfg

	in := &api.UserInput{
		Agent:   cfg.Agent,
		Command: cfg.Command,
	}

	var payload Payload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return StatusError, err.Error()
	}

	in.Content = payload.Content

	var messages []*api.Message
	for _, v := range payload.Parts {
		// TODO optimize and skip this step
		// LLM use the same encoding for multi media data
		// data:[<media-type>][;base64],<data>
		ca := strings.SplitN(v.Content, ",", 2)
		if len(ca) != 2 {
			return StatusError, "invalid multi media content"
		}
		raw, err := base64.StdEncoding.DecodeString(ca[1])
		if err != nil {
			return StatusError, err.Error()
		}
		messages = append(messages, &api.Message{
			ContentType: v.ContentType,
			Content:     string(raw),
			Role:        api.RoleUser,
		})
	}
	in.Messages = messages

	// run
	cfg.Format = "text"
	if err := agent.RunSwarm(cfg, in); err != nil {
		log.Errorf("error running agent: %s\n", err)
		return StatusError, err.Error()
	}

	//success
	log.Infof("ai executed successfully\n")

	return StatusOK, cfg.Stdout
}
