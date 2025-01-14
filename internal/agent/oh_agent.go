package agent

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"

	"github.com/qiangli/ai/internal/docker/oh"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
)

type OhAgent struct {
	config *llm.Config

	Role    string
	Message string
}

func NewOhAgent(cfg *llm.Config, role, content string) (*OhAgent, error) {
	if role == "" {
		role = "system"
	}
	if content == "" {
		content = resource.GetWSBaseSystemRoleContent()
	}

	cfg.Tools = llm.GetSystemTools()

	agent := OhAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func (r *OhAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	workspace, err := resolveWorkspaceBase(ctx, r.config, r.config.Workspace, input)
	if err != nil {
		return nil, err
	}

	log.Infof("using workspace: %s\n", workspace)

	// https://docs.all-hands.dev/modules/usage/how-to/headless-mode
	hostDir := workspace
	containerDir := oh.WorkspaceInSandbox
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
		Agent:   "OH",
		Content: content,
	}, nil
}
