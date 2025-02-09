package agent

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
)

var adviceMap = map[string]swarm.Advice{}

func decodeMetaResponseAdvice(_ *swarm.Request, resp *swarm.Response, next swarm.Advice) error {
	if next != nil {
		if err := next(nil, resp, nil); err != nil {
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

func scriptUserInput(req *swarm.Request, _ *swarm.Response, next swarm.Advice) error {
	cmd := req.Vars.Subcommand
	if cmd != "" {
		cmd = filepath.Base(cmd)
	}

	tpl, ok := resource.Prompts["script_user_role"]
	if !ok {
		return fmt.Errorf("no such prompt: script_user_role")
	}
	content, err := applyTemplate(tpl, map[string]interface{}{
		"Command": cmd,
		"Message": req.Vars.Input,
	})
	if err != nil {
		return err
	}
	req.Message = &swarm.Message{
		Role:    swarm.RoleUser,
		Content: content,
		Sender:  req.Vars.Agent,
	}

	if next != nil {
		return next(req, nil, nil)
	}
	return nil
}

func init() {
	adviceMap["decode_meta_response"] = decodeMetaResponseAdvice
	adviceMap["script_user_input"] = scriptUserInput
}
