package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/qiangli/ai/internal/agent/resource"
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/docker/aider"
	"github.com/qiangli/ai/internal/docker/gptr"
	"github.com/qiangli/ai/internal/docker/oh"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/swarm"
)

// builtin functions
var funcRegistry = map[string]swarm.Function{}

func init() {
	funcRegistry["gptr_generate_report"] = gptrGenerateReport
	funcRegistry["list_agents"] = listAgentFunc
	funcRegistry["agent_info"] = agentInfoFunc
}

func gptrGenerateReport(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	var obj gptr.ReportArgs

	b, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(b, &obj)
	if err != nil {
		return nil, err
	}

	content, err := GenerateReport(ctx, agent.Model, obj.ReportType, obj.Tone, agent.RawInput.Query())
	if err != nil {
		return nil, err
	}

	return &api.Result{
		Value: content,
		State: api.StateExit,
	}, nil
}

func GenerateReport(ctx context.Context, model *swarm.Model, reportType, tone, input string) (string, error) {
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

func Aider(ctx context.Context, models map[string]*swarm.Model, workspace, sub, input string) error {
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
	userContent, err := applyTemplate(tpl, &WSInput{
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
	model := models["L1"]

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

	os.Setenv("AIDER_WEAK_MODEL", models["L1"].Name)
	os.Setenv("AIDER_EDITOR_MODEL", models["L2"].Name)
	os.Setenv("AIDER_MODEL", models["L2"].Name)
	if sub == string(aider.Architect) {
		os.Setenv("AIDER_MODEL", models["L3"].Name)
	}

	os.Setenv("AIDER_VERBOSE", fmt.Sprintf("%v", log.IsVerbose()))

	return aider.Run(ctx, aider.ChatMode(sub), userContent)
}

func OpenHands(ctx context.Context, model *swarm.Model, workspace string, in *api.UserInput) error {
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
	userContent, err := applyTemplate(tpl, &WSInput{
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

func listAgentFunc(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	var list []string
	for k, v := range resource.AgentCommandMap {
		list = append(list, fmt.Sprintf("%s: %s", k, v.Description))
	}
	sort.Strings(list)
	return &api.Result{
		Value: fmt.Sprintf("Available agents:\n%s\n", strings.Join(list, "\n")),
	}, nil
}

func agentInfoFunc(ctx context.Context, agent *swarm.Agent, name string, args map[string]any) (*api.Result, error) {
	key, err := swarm.GetStrProp("agent", args)
	if err != nil {
		return nil, err
	}
	var result string
	if v, ok := resource.AgentCommandMap[key]; ok {
		result = v.Overview
		if result == "" {
			result = v.Description
		}
	}
	return &api.Result{
		Value: result,
	}, nil
}
