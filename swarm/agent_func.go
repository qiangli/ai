package swarm

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal/docker/aider"
	"github.com/qiangli/ai/internal/docker/gptr"
	"github.com/qiangli/ai/internal/docker/oh"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/swarm/agent/resource"
	"github.com/qiangli/ai/swarm/api"
)

// builtin functions
var funcRegistry = map[string]api.Function{}

const funcKitName = "ai"

func init() {
	funcRegistry["agent_transfer"] = agentTransferFunc
	funcRegistry["list_agents"] = listAgentFunc
	funcRegistry["agent_info"] = agentInfoFunc
}

var descriptors = []api.ToolDescriptor{
	{
		Name:        "list_agents",
		Description: "List all supported AI agents",
		Parameters:  nil,
	},
	{
		Name:        "agent_info",
		Description: "Get information about a specific agent",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent": map[string]any{
					"type":        "string",
					"description": "The name of the agent",
				},
			},
			"required": []any{"agent"},
		},
	},
	{
		Name:        "agent_transfer",
		Description: "Transfer the current task to a specific agent",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"agent": map[string]any{
					"type":        "string",
					"description": "The name of the agent",
				},
			},
			"required": []any{"agent"},
		},
	},
}

func ListFuncTools() ([]*api.ToolFunc, error) {
	var tools []*api.ToolFunc
	for _, desc := range descriptors {
		tools = append(tools, &api.ToolFunc{
			Type:        ToolTypeFunc,
			Kit:         funcKitName,
			Name:        desc.Name,
			Description: desc.Description,
			Parameters:  desc.Parameters,
		})
	}
	return tools, nil
}

func GenerateReport(ctx context.Context, model *api.Model, reportType, tone, input string) (string, error) {
	// FIXME: This is a hack
	// better to config the base url and api key (and others) for gptr
	u, err := url.Parse(model.BaseUrl)
	if err != nil {
		return "", err
	}

	if isLoopback(u.Host) {
		_, port, _ := net.SplitHostPort(u.Host)
		u.Host = "host.docker.internal:" + port
	}
	os.Setenv("OPENAI_API_BASE", u.String())
	os.Setenv("OPENAI_API_KEY", model.ApiKey)

	tempDir, err := os.MkdirTemp("", "gptr")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	err = gptr.GenerateReport(ctx, reportType, tone, input, tempDir)
	if err != nil {
		return "", err
	}

	return readReport(tempDir)
}

func readReport(tempDir string) (string, error) {
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".md") {
			filePath := filepath.Join(tempDir, file.Name())
			content, err := os.ReadFile(filePath)
			if err != nil {
				return "", err
			}
			return string(content), nil
		}
	}
	return "", fmt.Errorf("no report file generated")
}

type WSInput struct {
	Env          string
	HostDir      string
	ContainerDir string
	Input        string
}

func Aider(ctx context.Context, models map[api.Level]*api.Model, workspace, sub, input string) error {
	log.Infof("using workspace: %s\n", workspace)

	if sub == "" {
		sub = string(aider.Code)
	}
	if workspace == "" {
		return fmt.Errorf("workspace is required")
	}

	// https://docs.all-hands.dev/modules/usage/how-to/headless-mode
	hostDir := workspace
	containerDir := aider.WorkspaceInSandbox
	env := "container"

	tpl, ok := resource.Prompts["docker_input_user_role"]
	if !ok {
		return fmt.Errorf("no such prompt: docker_input_user_role")
	}
	userContent, err := applyDefaultTemplate(tpl, &WSInput{
		Env:          env,
		HostDir:      hostDir,
		ContainerDir: containerDir,
		Input:        input,
	})
	if err != nil {
		return err
	}

	// Set the workspace
	os.Setenv("WORKSPACE_BASE", workspace)

	// Calling out to OH
	// FIXME: This is a hack
	// better to config the base url and api key (and others) for oh

	// FIXME this wont work if model providers are different
	model := models[api.L1]

	u, err := url.Parse(model.BaseUrl)
	if err != nil {
		return fmt.Errorf("failed to parse base url: %w", err)
	}

	if isLoopback(u.Host) {
		_, port, _ := net.SplitHostPort(u.Host)
		u.Host = "host.docker.internal:" + port
	}

	os.Setenv("OPENAI_API_BASE", u.String())
	os.Setenv("OPENAI_API_KEY", model.ApiKey)

	os.Setenv("AIDER_WEAK_MODEL", models[api.L1].Name)
	os.Setenv("AIDER_EDITOR_MODEL", models[api.L2].Name)
	os.Setenv("AIDER_MODEL", models[api.L2].Name)
	if sub == string(aider.Architect) {
		os.Setenv("AIDER_MODEL", models[api.L3].Name)
	}

	os.Setenv("AIDER_VERBOSE", fmt.Sprintf("%v", log.IsVerbose()))

	return aider.Run(ctx, aider.ChatMode(sub), userContent)
}

func OpenHands(ctx context.Context, model *api.Model, workspace string, in *api.UserInput) error {
	log.Infof("using workspace: %s\n", workspace)

	if workspace == "" {
		return fmt.Errorf("workspace is required")
	}

	// https://docs.all-hands.dev/modules/usage/how-to/headless-mode
	hostDir := workspace
	containerDir := oh.WorkspaceInSandbox
	env := "container"

	tpl, ok := resource.Prompts["docker_input_user_role"]
	if !ok {
		return fmt.Errorf("no such prompt: docker_input_user_role")
	}
	userContent, err := applyDefaultTemplate(tpl, &WSInput{
		Env:          env,
		HostDir:      hostDir,
		ContainerDir: containerDir,
		Input:        in.Query(),
	})
	if err != nil {
		return err
	}

	// Set the workspace
	os.Setenv("WORKSPACE_BASE", workspace)
	os.Setenv("WORKSPACE_MOUNT_PATH", workspace)
	os.Setenv("WORKSPACE_MOUNT_PATH_IN_SANDBOX", oh.WorkspaceInSandbox)

	// Calling out to OH
	// FIXME: This is a hack
	// better to config the base url and api key (and others) for oh
	u, err := url.Parse(model.BaseUrl)
	if err != nil {
		return err
	}

	if isLoopback(u.Host) {
		_, port, _ := net.SplitHostPort(u.Host)
		u.Host = "host.docker.internal:" + port
	}
	os.Setenv("LLM_BASE_URL", u.String())
	os.Setenv("LLM_API_KEY", model.ApiKey)
	os.Setenv("LLM_MODEL", model.Name)
	os.Setenv("DEBUG", fmt.Sprintf("%v", log.IsVerbose()))

	return oh.Run(ctx, userContent)
}

func listAgentFunc(ctx context.Context, _ *api.Vars, _ string, _ map[string]any) (*api.Result, error) {
	var list []string
	for k, v := range resource.AgentCommandMap {
		list = append(list, fmt.Sprintf("%s: %s", k, v.Description))
	}

	sort.Strings(list)
	return &api.Result{
		Value: fmt.Sprintf("Available agents:\n%s\n", strings.Join(list, "\n")),
	}, nil
}

func agentInfoFunc(ctx context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	var result string
	if v, ok := resource.AgentCommandMap[agent]; ok {
		result = v.Overview
		if result == "" {
			result = v.Description
		}
	} else {
		return nil, fmt.Errorf("unknown agent: %s", agent)
	}
	return &api.Result{
		Value: result,
	}, nil
}

func agentTransferFunc(ctx context.Context, _ *api.Vars, _ string, args map[string]any) (*api.Result, error) {
	agent, err := GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	if _, ok := resource.AgentCommandMap[agent]; !ok {
		return nil, fmt.Errorf("unknown agent: %s", agent)
	}
	return &api.Result{
		NextAgent: agent,
		State:     api.StateTransfer,
	}, nil
}
