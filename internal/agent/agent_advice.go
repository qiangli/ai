package agent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/agent/resource/pr"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
)

var adviceMap = map[string]swarm.Advice{}

func init() {
	adviceMap["decode_meta_response"] = decodeMetaResponseAdvice
	adviceMap["script_user_input"] = scriptUserInputAdvice
	adviceMap["pr_user_input"] = prUserInputAdvice
	adviceMap["pr_json_to_markdown"] = prFormatAdvice
	adviceMap["resolve_workspace"] = resolveWorkspaceAdvice
	adviceMap["aider"] = aiderAdvice
	adviceMap["openhands"] = ohAdvice
	adviceMap["agent_launch"] = agentLaunchAdvice
}

type AgentDetect struct {
	Agent   string `json:"agent"`
	Command string `json:"command"`
}

// agent after advice
func agentLaunchAdvice(vars *swarm.Vars, req *swarm.Request, resp *swarm.Response, _ swarm.Advice) error {
	var v AgentDetect
	msg := resp.LastMessage()
	if msg == nil {
		return fmt.Errorf("invalid response: no message")
	}

	if err := json.Unmarshal([]byte(msg.Content), &v); err != nil {
		log.Debugf("decode_meta_response error: %v", err)
		return nil
	}

	//
	req.RawInput.Subcommand = v.Command
	//
	resp.Transfer = true
	resp.NextAgent = v.Agent

	log.Debugf("dispatching: %+v\n", v)

	return nil
}

type metaResponse struct {
	Service    string `json:"service"`
	RolePrompt string `json:"system_role_prompt"`
}

// meta prompt after advice
func decodeMetaResponseAdvice(vars *swarm.Vars, _ *swarm.Request, resp *swarm.Response, _ swarm.Advice) error {
	var v metaResponse
	msg := resp.LastMessage()
	if msg == nil {
		return fmt.Errorf("invalid response: no message")
	}

	if err := json.Unmarshal([]byte(msg.Content), &v); err != nil {
		log.Debugf("decode_meta_response error: %v", err)
		return nil
	}

	vars.Extra["service"] = v.Service
	vars.Extra["system_role_prompt"] = v.RolePrompt

	log.Debugf("decode_meta_response: %+v\n", v)

	return nil
}

// script user input before advice
func scriptUserInputAdvice(vars *swarm.Vars, req *swarm.Request, _ *swarm.Response, _ swarm.Advice) error {
	in := req.RawInput

	cmd := in.Subcommand
	if cmd != "" {
		cmd = filepath.Base(cmd)
	}
	tpl, ok := resource.Prompts["script_user_role"]
	if !ok {
		return fmt.Errorf("no such prompt: script_user_role")
	}
	content, err := applyTemplate(tpl, map[string]any{
		"Command": cmd,
		"Message": in.Query(),
	})
	if err != nil {
		return err
	}
	req.Message = &swarm.Message{
		Role:    swarm.RoleUser,
		Content: content,
		Sender:  req.Agent,
	}

	return nil
}

// PR user input before advice
func prUserInputAdvice(vars *swarm.Vars, req *swarm.Request, _ *swarm.Response, _ swarm.Advice) error {
	in := req.RawInput

	tpl, ok := resource.Prompts["pr_user_role"]
	if !ok {
		return fmt.Errorf("no such prompt: script_user_role")
	}

	data := map[string]any{
		"instruction": in.Message,
		"diff":        in.Content,
		"changelog":   "", // TODO: add changelog
		"today":       time.Now().Format("2006-01-02"),
	}
	content, err := applyTemplate(tpl, data)
	if err != nil {
		return err
	}
	req.Message = &swarm.Message{
		Role:    swarm.RoleUser,
		Content: content,
		Sender:  req.Agent,
	}

	return nil
}

// PR format after advice
func prFormatAdvice(vars *swarm.Vars, req *swarm.Request, resp *swarm.Response, _ swarm.Advice) error {
	name := req.Agent
	var tplName = fmt.Sprintf("pr_%s_format", name)
	tpl, ok := resource.Prompts[tplName]
	if !ok {
		return fmt.Errorf("no such prompt resource: %s", tplName)
	}

	formatPrDescription := func(resp string) (string, error) {
		var data pr.PRDescription
		if err := tryUnmarshal(resp, &data); err != nil {
			return "", fmt.Errorf("error unmarshaling response: %w", err)
		}
		return applyTemplate(tpl, &data)
	}
	formatPrCodeSuggestion := func(resp string) (string, error) {
		var data pr.PRCodeSuggestions
		if err := tryUnmarshal(resp, &data); err != nil {
			return "", fmt.Errorf("error unmarshaling response: %w", err)
		}
		return applyTemplate(tpl, data.CodeSuggestions)
	}
	formatPrReview := func(resp string) (string, error) {
		var data pr.PRReview
		if err := tryUnmarshal(resp, &data); err != nil {
			return "", fmt.Errorf("error unmarshaling response: %w", err)
		}
		return applyTemplate(tpl, &data.Review)
	}
	formatPrChangelog := func(resp string) (string, error) {
		return applyTemplate(tpl, &pr.PRChangelog{
			Changelog: resp,
			Today:     time.Now().Format("2006-01-02"),
		})
	}

	msg := resp.LastMessage()
	var content = msg.Content
	var err error
	switch name {
	case "describe":
		content, err = formatPrDescription(content)
	case "review":
		content, err = formatPrReview(content)
	case "improve":
		content, err = formatPrCodeSuggestion(content)
	case "changelog":
		content, err = formatPrChangelog(content)
	default:
		return fmt.Errorf("unknown agent/subcommand: %s", name)
	}
	if err != nil {
		return err
	}
	msg.Content = content

	return nil
}

func aiderAdvice(vars *swarm.Vars, req *swarm.Request, resp *swarm.Response, _ swarm.Advice) error {
	return Aider(req.Context(), vars.Models, vars.Workspace, req.RawInput.Subcommand, req.RawInput.Query())
}

func ohAdvice(vars *swarm.Vars, req *swarm.Request, resp *swarm.Response, _ swarm.Advice) error {
	return OpenHands(req.Context(), vars.Models["L2"], vars.Workspace, req.RawInput)
}
