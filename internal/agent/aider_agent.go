package agent

// import (
// 	"context"
// 	"fmt"
// 	"net"
// 	"net/url"
// 	"os"

// 	"github.com/qiangli/ai/internal"
// 	"github.com/qiangli/ai/internal/docker/aider"
// 	"github.com/qiangli/ai/internal/log"
// 	"github.com/qiangli/ai/internal/resource"
// )

// type AiderAgent struct {
// 	config *internal.AppConfig

// 	// Role   string
// 	// Prompt string
// }

// func NewAiderAgent(cfg *internal.AppConfig) (*AiderAgent, error) {
// 	// role := cfg.Role
// 	// // prompt := cfg.Prompt

// 	// if role == "" {
// 	// 	role = "system"
// 	// }

// 	agent := AiderAgent{
// 		config: cfg,
// 		// Role:   role,
// 		// Prompt: prompt,
// 	}
// 	return &agent, nil
// }

// func (r *AiderAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
// 	log.Debugf(`AiderAgent.Send: subcommand: %s\n`, in.Subcommand)
// 	if in.Subcommand == "" {
// 		in.Subcommand = string(aider.Code)
// 	}

// 	var input = in.Input()

// 	workspace, err := resolveWorkspaceBase(ctx, r.config.LLM, r.config.LLM.Workspace, in.Intent())
// 	if err != nil {
// 		return nil, err
// 	}

// 	log.Infof("using workspace: %s\n", workspace)

// 	// https://docs.all-hands.dev/modules/usage/how-to/headless-mode
// 	hostDir := workspace
// 	containerDir := aider.WorkspaceInSandbox
// 	env := "container"
// 	userContent, err := resource.GetWSEnvContextInput(&resource.WSInput{
// 		Env:          env,
// 		HostDir:      hostDir,
// 		ContainerDir: containerDir,
// 		Input:        input,
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	// Set the workspace
// 	r.config.LLM.WorkDir = workspace
// 	os.Setenv("WORKSPACE_BASE", workspace)

// 	// Calling out to OH
// 	// FIXME: This is a hack
// 	// better to config the base url and api key (and others) for oh
// 	u, err := url.Parse(r.config.LLM.BaseUrl)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to parse base url: %w", err)
// 	}

// 	if isLoopback(u.Host) {
// 		_, port, _ := net.SplitHostPort(u.Host)
// 		u.Host = "host.docker.internal:" + port
// 	}
// 	os.Setenv("OPENAI_API_BASE", u.String())
// 	os.Setenv("OPENAI_API_KEY", r.config.LLM.ApiKey)

// 	os.Setenv("AIDER_WEAK_MODEL", r.config.LLM.L1Model)
// 	os.Setenv("AIDER_EDITOR_MODEL", r.config.LLM.L2Model)
// 	os.Setenv("AIDER_MODEL", r.config.LLM.L2Model)
// 	if in.Subcommand == string(aider.Architect) {
// 		os.Setenv("AIDER_MODEL", r.config.LLM.L3Model)
// 	}

// 	os.Setenv("AIDER_VERBOSE", fmt.Sprintf("%v", r.config.LLM.Debug))

// 	var content string

// 	if !internal.DryRun {
// 		err = aider.Run(ctx, aider.ChatMode(in.Subcommand), userContent)
// 		if err != nil {
// 			return nil, err
// 		}
// 	} else {
// 		content = internal.DryRunContent
// 	}

// 	return &ChatMessage{
// 		Agent:   "AIDER",
// 		Content: content,
// 	}, nil
// }
