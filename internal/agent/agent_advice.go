package agent

import (
	"encoding/json"
	"fmt"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
)

var adviceMap = map[string]swarm.Advice{}

func decodeMetaResponseAdvice(req *swarm.Request, resp *swarm.Response, next swarm.Advice) error {
	if next != nil {
		if err := next(req, resp, nil); err != nil {
			return err
		}
	}

	if resp == nil {
		return nil
	}

	var v AskAgentPrompt
	msg := resp.LastMessage()
	if msg == nil {
		return fmt.Errorf("invalid response: no message")
	}

	if err := json.Unmarshal([]byte(msg.Content), &v); err != nil {
		log.Debugf("decode_meta_response error: %v", err)
		return nil
	}

	resp.AddExtra("service", v.Service)
	resp.AddExtra("system_role_prompt", v.RolePrompt)

	log.Debugf("decode_meta_response: %+v\n", v)

	return nil
}

func init() {
	adviceMap["decode_meta_response"] = decodeMetaResponseAdvice
}
