package hub

import (
	"encoding/base64"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/qiangli/ai/agent"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
)

func triggerAI(content string) (string, bool) {
	const prefix = ""
	const triggerWord = "ai"
	re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(prefix) + `\s*(?i:` + regexp.QuoteMeta(triggerWord) + `)\s+(.*)`)
	match := re.FindStringSubmatch(content)
	if len(match) < 2 {
		return content, false
	}
	return match[1], true
}

func (h *Hub) handleAI(data string) (ActionStatus, string) {
	log.Debugf("handle AI: size %v\n", len(data))

	if h.cfg == nil {
		h.cfg = &api.AppConfig{}
	}
	cfg := h.cfg.Clone()

	// default to text for chatbots
	cfg.Format = "text"

	var payload Payload
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return StatusError, err.Error()
	}

	var content = payload.Content
	// detect trigger "ai" - case insensitive
	if line, ok := triggerAI(content); ok {
		log.Debugf("embedded ai command found: %s\n", line)
		if err := parseFlags(line, cfg); err != nil {
			return StatusError, err.Error()
		}
		content = strings.Join(cfg.Args, " ")
		if cfg.New {
			cfg.History = nil
		}
		// TODO other cfg may require updates
	}

	in := &api.UserInput{
		Agent:   cfg.Agent,
		Command: cfg.Command,
		Content: content,
	}

	log.Debugf("new cfg: %+v\n", cfg)
	log.Debugf("content: %s\n", content)

	var messages []*api.Message
	for _, v := range payload.Parts {
		if strings.HasPrefix(v.ContentType, "text/") {
			messages = append(messages, &api.Message{
				ContentType: v.ContentType,
				Content:     v.Content,
				Role:        api.RoleUser,
			})
			continue
		}
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
	if err := agent.RunSwarm(cfg, in); err != nil {
		log.Errorf("error running agent: %s\n", err)
		return StatusError, err.Error()
	}

	//success
	log.Infof("ai executed successfully\n")

	return StatusOK, cfg.Stdout
}
