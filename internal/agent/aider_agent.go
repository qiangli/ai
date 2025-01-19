package agent

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/docker/aider"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
)

type AiderAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string
}

func NewAiderAgent(cfg *internal.AppConfig) (*AiderAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt

	if role == "" {
		role = "system"
	}

	cfg.LLM.Tools = llm.GetSystemTools()

	agent := AiderAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &agent, nil
}

func (r *AiderAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	var input = in.Input()
	var clip = in.Clip()

	workspace, err := resolveWorkspaceBase(ctx, r.config.LLM, r.config.LLM.Workspace, clip)
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
	r.config.LLM.WorkDir = workspace
	os.Setenv("WORKSPACE_BASE", workspace)

	// Calling out to OH
	// FIXME: This is a hack
	// better to config the base url and api key (and others) for oh
	u, err := url.Parse(r.config.LLM.BaseUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse base url: %w", err)
	}

	if isLoopback(u.Host) {
		_, port, _ := net.SplitHostPort(u.Host)
		u.Host = "host.docker.internal:" + port
	}
	os.Setenv("OPENAI_API_BASE", u.String())
	os.Setenv("OPENAI_API_KEY", r.config.LLM.ApiKey)
	os.Setenv("AIDER_MODEL", r.config.LLM.L2Model)
	os.Setenv("AIDER_WEAK_MODEL", r.config.LLM.L1Model)

	os.Setenv("AIDER_VERBOSE", fmt.Sprintf("%v", r.config.LLM.Debug))

	var content string

	if !r.config.LLM.DryRun {
		err = aider.Run(ctx, aider.Code, userContent)
		if err != nil {
			return nil, err
		}
	} else {
		content = r.config.LLM.DryRunContent
	}

	return &ChatMessage{
		Agent:   "AIDER",
		Content: content,
	}, nil
}
