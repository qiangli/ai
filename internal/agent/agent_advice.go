package agent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/agent/resource/pr"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/llm"
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
	adviceMap["sub"] = subAdvice
	adviceMap["image_params"] = imageParamsAdvice
	adviceMap["chdir_format_path"] = chdirFormatPathAdvice
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
	req.RawInput.Command = v.Command
	//
	resp.Result = &swarm.Result{
		State:     api.StateTransfer,
		NextAgent: v.Agent,
	}

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

	cmd := in.Command
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
		Role:    api.RoleUser,
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
		Role:    api.RoleUser,
		Content: content,
		Sender:  req.Agent,
	}

	return nil
}

// PR format after advice
func prFormatAdvice(vars *swarm.Vars, req *swarm.Request, resp *swarm.Response, _ swarm.Advice) error {
	name := baseCommand(req.Agent)
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
	return Aider(req.Context(), vars.Models, vars.Workspace, req.RawInput.Command, req.RawInput.Query())
}

func ohAdvice(vars *swarm.Vars, req *swarm.Request, resp *swarm.Response, _ swarm.Advice) error {
	return OpenHands(req.Context(), vars.Models[api.L2], vars.Workspace, req.RawInput)
}

func subAdvice(vars *swarm.Vars, req *swarm.Request, resp *swarm.Response, next swarm.Advice) error {
	sub := baseCommand(req.RawInput.Command)
	if sub != "" {
		resp.Result = &swarm.Result{
			State:     api.StateTransfer,
			NextAgent: fmt.Sprintf("%s/%s", req.Agent, sub),
		}
		return nil
	}
	return next(vars, req, resp, next)
}

type ImageParams struct {
	Quality string `json:"quality"`
	Size    string `json:"size"`
	Style   string `json:"style"`
}

func imageParamsAdvice(vars *swarm.Vars, req *swarm.Request, resp *swarm.Response, next swarm.Advice) error {
	// skip if all image params are already set
	if req.ImageQuality != "" && req.ImageSize != "" && req.ImageStyle != "" {
		log.Debugf("skip image params advice as all are already set")
		return nil
	}

	model, ok := vars.Models[api.L1]
	if !ok {
		log.Debugf("no model found")
		return nil
	}
	ctx := req.Context()
	var msgs = []*api.Message{
		{
			Role:    api.RoleSystem,
			Content: resource.Prompts["image_param_system_role"],
		},
		{
			Role:    api.RoleUser,
			Content: req.RawInput.Intent(),
		},
	}

	result, err := llm.Send(ctx, &api.Request{
		ModelType: model.Type,
		BaseUrl:   model.BaseUrl,
		ApiKey:    model.ApiKey,
		Model:     model.Name,
		Messages:  msgs,
	})
	if err != nil {
		log.Errorf("error sending request: %v", err)
		return nil
	}

	var params ImageParams
	if err := json.Unmarshal([]byte(result.Content), &params); err != nil {
		log.Debugf("error unmarshaling response: %v", err)
		return nil
	}

	if params.Quality == "" {
		req.ImageQuality = params.Quality
	}
	if params.Size == "" {
		req.ImageSize = params.Size
	}
	if params.Style == "" {
		req.ImageStyle = params.Style
	}
	return nil
}

// chdir format path after advice
func chdirFormatPathAdvice(vars *swarm.Vars, _ *swarm.Request, resp *swarm.Response, _ swarm.Advice) error {
	var v struct {
		Action    string `json:"action"`
		Success   bool   `json:"success"`
		Directory string `json:"directory"`
	}

	msg := resp.LastMessage()
	if msg == nil {
		return fmt.Errorf("invalid response: no message")
	}

	// don't change directory if action is not "chdir" or if it was not successful
	if err := json.Unmarshal([]byte(msg.Content), &v); err != nil {
		log.Debugf("chdir_format_path error: %v", err)
		msg.Content = "./"
		return nil
	}
	if v.Action != "chdir" || !v.Success {
		msg.Content = "./"
		return nil
	}
	msg.Content = v.Directory

	log.Debugf("chdir_format_path: %+v\n", v)

	return nil
}
