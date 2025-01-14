package agent

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/qiangli/ai/internal/docker/aider"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
)

type AiderAgent struct {
	config *llm.Config

	Role    string
	Message string
}

func NewAiderAgent(cfg *llm.Config, role, content string) (*AiderAgent, error) {
	if role == "" {
		role = "system"
	}

	cfg.Tools = llm.GetSystemTools()

	agent := AiderAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *AiderAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	workspace, err := resolveWorkspaceBase(ctx, r.config, r.config.Workspace, input)
	if err != nil {
		return nil, err
	}

	log.Infof("using workspace: %s\n", workspace)

	// https://docs.all-hands.dev/modules/usage/how-to/headless-mode
	hostDir := workspace
	containerDir := aider.WorkspaceInSandbox
	env := "container"
	userContent, err := resource.GetWSEnvContextInput(&resource.WSInput{
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
	os.Setenv("WORKSPACE_BASE", workspace)

	// Calling out to OH
	// FIXME: This is a hack
	// better to config the base url and api key (and others) for oh
	u, err := url.Parse(r.config.BaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base url: %w", err)
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
		Agent:   "AIDER",
		Content: content,
	}, nil
}
