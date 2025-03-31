package swarm

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/api"
	"github.com/qiangli/ai/swarm/llm"
	pr "github.com/qiangli/ai/swarm/resource/agents/pr"
)

var adviceMap = map[string]api.Advice{}

func init() {
	// adviceMap["decode_meta_response"] = decodeMetaResponseAdvice
	adviceMap["script_user_input"] = scriptUserInputAdvice
	adviceMap["pr_user_input"] = prUserInputAdvice
	adviceMap["pr_json_to_markdown"] = prFormatAdvice
	adviceMap["resolve_workspace"] = resolveWorkspaceAdvice
	adviceMap["aider"] = aiderAdvice
	adviceMap["openhands"] = ohAdvice
	// adviceMap["agent_launch"] = agentLaunchAdvice
	adviceMap["sub"] = subAdvice
	adviceMap["image_params"] = imageParamsAdvice
	adviceMap["chdir_format_path"] = chdirFormatPathAdvice
}

// script user input before advice
func scriptUserInputAdvice(vars *api.Vars, req *api.Request, _ *api.Response, _ api.Advice) error {
	in := req.RawInput

	cmd := in.Command
	if cmd != "" {
		cmd = filepath.Base(cmd)
	}

	tpl, err := vars.Resource(req.Agent, "script_user_role.md")
	if err != nil {
		return err
	}
	content, err := applyDefaultTemplate(string(tpl), map[string]any{
		"Command": cmd,
		"Message": in.Query(),
	})
	if err != nil {
		return err
	}
	req.Message = &api.Message{
		Role:    api.RoleUser,
		Content: content,
		Sender:  req.Agent,
	}

	return nil
}

// PR user input before advice
func prUserInputAdvice(vars *api.Vars, req *api.Request, _ *api.Response, _ api.Advice) error {
	tpl, err := vars.Resource(req.Agent, "pr_user_role.md")
	if err != nil {
		return err
	}

	in := req.RawInput
	data := map[string]any{
		"instruction": in.Message,
		"diff":        in.Content,
		"changelog":   "", // TODO: add changelog
		"today":       time.Now().Format("2006-01-02"),
	}
	content, err := applyDefaultTemplate(string(tpl), data)
	if err != nil {
		return err
	}
	req.Message = &api.Message{
		Role:    api.RoleUser,
		Content: content,
		Sender:  req.Agent,
	}

	return nil
}

// PR format after advice
func prFormatAdvice(vars *api.Vars, req *api.Request, resp *api.Response, _ api.Advice) error {
	cmd := req.Command
	if cmd == "" {
		parts := strings.SplitN(req.Agent, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid agent: %s", req.Agent)
		}
		cmd = parts[1]
	}
	var tplName = fmt.Sprintf("pr_%s_format.md", cmd)
	b, err := vars.Resource(req.Agent, tplName)
	if err != nil {
		return err
	}
	tpl := string(b)

	formatPrDescription := func(resp string) (string, error) {
		var data pr.PRDescription
		if err := tryUnmarshal(resp, &data); err != nil {
			return "", fmt.Errorf("error unmarshaling response: %w", err)
		}
		return applyDefaultTemplate(string(tpl), &data)
	}
	formatPrCodeSuggestion := func(resp string) (string, error) {
		var data pr.PRCodeSuggestions
		if err := tryUnmarshal(resp, &data); err != nil {
			return "", fmt.Errorf("error unmarshaling response: %w", err)
		}
		return applyDefaultTemplate(tpl, data.CodeSuggestions)
	}
	formatPrReview := func(resp string) (string, error) {
		var data pr.PRReview
		if err := tryUnmarshal(resp, &data); err != nil {
			return "", fmt.Errorf("error unmarshaling response: %w", err)
		}
		return applyDefaultTemplate(tpl, &data.Review)
	}
	formatPrChangelog := func(resp string) (string, error) {
		return applyDefaultTemplate(tpl, &pr.PRChangelog{
			Changelog: resp,
			Today:     time.Now().Format("2006-01-02"),
		})
	}

	msg := resp.LastMessage()
	var content = msg.Content
	// var err error
	switch cmd {
	case "describe":
		content, err = formatPrDescription(content)
	case "review":
		content, err = formatPrReview(content)
	case "improve":
		content, err = formatPrCodeSuggestion(content)
	case "changelog":
		content, err = formatPrChangelog(content)
	default:
		return fmt.Errorf("unknown agent command: %s for %s", cmd, req.Agent)
	}
	if err != nil {
		return err
	}
	msg.Content = content

	return nil
}

func aiderAdvice(vars *api.Vars, req *api.Request, resp *api.Response, _ api.Advice) error {
	return Aider(req.Context(), vars.Models, vars.Workspace, req.RawInput.Command, req.RawInput.Query())
}

func ohAdvice(vars *api.Vars, req *api.Request, resp *api.Response, _ api.Advice) error {
	return OpenHands(req.Context(), vars.Models[api.L2], vars.Workspace, req.RawInput)
}

// subAdvice is an around advice that checks if a subcommand is specified.
// skip LLM if it is and go directly to the next sub agent.
func subAdvice(vars *api.Vars, req *api.Request, resp *api.Response, next api.Advice) error {
	sub := baseCommand(req.RawInput.Command)
	if sub != "" {
		resp.Result = &api.Result{
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

func imageParamsAdvice(vars *api.Vars, req *api.Request, resp *api.Response, next api.Advice) error {
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

	content, err := vars.Resource(req.Agent, "image_param_system_role.md")
	if err != nil {
		return err
	}

	var msgs = []*api.Message{
		{
			Role:    api.RoleSystem,
			Content: string(content),
		},
		{
			Role:    api.RoleUser,
			Content: req.RawInput.Intent(),
		},
	}

	result, err := llm.Send(ctx, &api.LLMRequest{
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
func chdirFormatPathAdvice(vars *api.Vars, _ *api.Request, resp *api.Response, _ api.Advice) error {
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

const missingWorkspace = "Please specify a workspace base directory."

type WorkspaceCheck struct {
	WorkspaceBase string `json:"workspace_base"`
	Detected      bool   `json:"detected"`
}

// resolveWorkspaceAdvice resolves the workspace base path.
// Detect the workspace from the input using LLM.
func resolveWorkspaceAdvice(vars *api.Vars, req *api.Request, resp *api.Response, next api.Advice) error {
	tpl, err := vars.Resource(req.Agent, "workspace_user_role.md")
	if err != nil {
		return err
	}

	query, err := applyDefaultTemplate(string(tpl), map[string]string{
		"Input": req.RawInput.Intent(),
	})
	if err != nil {
		return err
	}

	msg := &api.Message{
		Role:    api.RoleUser,
		Content: query,
		Sender:  req.Agent,
	}
	req.Message = msg
	if err := next(vars, req, resp, next); err != nil {
		return err
	}
	result := resp.LastMessage()

	var wsCheck WorkspaceCheck
	if err := json.Unmarshal([]byte(result.Content), &wsCheck); err != nil {
		return fmt.Errorf("%s: %w", missingWorkspace, err)
	}
	if !wsCheck.Detected {
		return fmt.Errorf("%s", missingWorkspace)
	}

	log.Debugf("workspace check: %+v\n", wsCheck)

	workspace := wsCheck.WorkspaceBase

	log.Infof("Workspace to use: %s\n", workspace)

	vars.Extra["workspace_base"] = workspace

	return nil
}
