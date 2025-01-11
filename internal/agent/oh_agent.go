package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/qiangli/ai/internal/docker/oh"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
	"github.com/qiangli/ai/internal/tool"
)

type OhAgent struct {
	config *llm.Config

	Role    string
	Message string
}

type WorkspaceCheck struct {
	WorkspaceBase string `json:"workspace_base"`
	IsValid       bool   `json:"is_valid"`
	Exist         bool   `json:"exist"`
	Reason        string `json:"reason"`
}

func NewOhAgent(cfg *llm.Config, role, content string) (*OhAgent, error) {
	if role == "" {
		role = "system"
	}
	if content == "" {
		content = resource.GetWSCheckSystemRoleContent()
	}

	cfg.Tools = tool.SystemTools

	agent := OhAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *OhAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	const myName string = "OH"

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
	containerDir := oh.WorkspaceInSandbox
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
	os.Setenv("WORKSPACE_MOUNT_PATH", workspace)
	os.Setenv("WORKSPACE_MOUNT_PATH_IN_SANDBOX", oh.WorkspaceInSandbox)

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
	os.Setenv("LLM_BASE_URL", u.String())
	os.Setenv("LLM_API_KEY", r.config.ApiKey)
	os.Setenv("LLM_MODEL", r.config.Model)
	os.Setenv("DEBUG", fmt.Sprintf("%v", r.config.Debug))

	var content string

	if !r.config.DryRun {
		err = oh.Run(ctx, userContent)
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
