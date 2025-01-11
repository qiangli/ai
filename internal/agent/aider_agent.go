package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/qiangli/ai/internal/docker/aider"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/tool"
)

type AiderAgent struct {
	config *llm.Config

	Role    string
	Message string
}

// type WorkspaceCheck struct {
// 	WorkspaceBase string `json:"workspace_base"`
// 	IsValid       bool   `json:"is_valid"`
// 	Exist         bool   `json:"exist"`
// 	Reason        string `json:"reason"`
// }

func NewAiderAgent(cfg *llm.Config, role, content string) (*AiderAgent, error) {
	if role == "" {
		role = "system"
	}
	if content == "" {
		content = resource.GetWSCheckSystemRoleContent()
	}

	cfg.Tools = tool.SystemTools

	agent := AiderAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *AiderAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	const myName string = "AIDER"

	// Decide the workspace with the help from LLM
	userContent, err := resource.GetWSCheckUserRoleContent(
		input,
	)
	if err != nil {
		return nil, err
	}
	resp, err := llm.Send(r.config, ctx, r.Role, r.Message, userContent)
	if err != nil {
		return nil, err
	}
	// unmarshal the response
	// TODO: retry?
	var wsCheck WorkspaceCheck
	err = json.Unmarshal([]byte(resp), &wsCheck)
	if err != nil {
		return nil, err
	}
	if !wsCheck.IsValid {
		return &ChatMessage{
			Agent:   myName,
			Content: wsCheck.Reason,
		}, nil
	}

	log.Debugf("Workspace check: %+v\n", wsCheck)

	// TODO: double check the workspace it is valid
	workspace := wsCheck.WorkspaceBase

	// https://docs.all-hands.dev/modules/usage/how-to/headless-mode
	hostDir := workspace
	containerDir := aider.WorkspaceInSandbox
	env := "container"
	userContent, err = resource.GetWSUserInputInstruction(&resource.WSInput{
		Env:          env,
		HostDir:      hostDir,
		ContainerDir: containerDir,
		Input:        input,
	})
	if err != nil {
		return nil, err
	}

	// Set the workspace
	r.config.WorkDir = workspace
	if !wsCheck.Exist {
		if err := os.MkdirAll(workspace, 0755); err != nil {
			return nil, err
		}
	}
	os.Setenv("WORKSPACE_BASE", workspace)

	// Calling out to OH
	// FIXME: This is a hack
	// better to config the base url and api key (and others) for oh
	u, err := url.Parse(r.config.BaseUrl)
	if err != nil {
		return nil, err
	}

	if isLoopback(u.Host) {
		_, port, _ := net.SplitHostPort(u.Host)
		u.Host = "host.docker.internal:" + port
	}
	os.Setenv("OPENAI_API_BASE", u.String())
	os.Setenv("OPENAI_API_KEY", r.config.ApiKey)
	os.Setenv("AIDER_MODEL", r.config.L2Model)
	os.Setenv("AIDER_WEAK_MODEL", r.config.L1Model)

	os.Setenv("AIDER_VERBOSE", fmt.Sprintf("%v", r.config.Debug))

	var content string

	if !r.config.DryRun {
		err = aider.Run(ctx, aider.Code, userContent)
		if err != nil {
			return nil, err
		}
	} else {
		content = r.config.DryRunContent
	}

	return &ChatMessage{
		Agent:   myName,
		Content: content,
	}, nil
}
