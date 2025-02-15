package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/docker/gptr"
	"github.com/qiangli/ai/internal/swarm"
)

var funcRegistry = map[string]swarm.Function{}

func init() {
	funcRegistry["gptr_generate_report"] = gptrGenerateReport
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

	content, err := GenerateReport(ctx, agent.Model, obj.ReportType, obj.Tone, agent.Vars.Input.Input())
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
