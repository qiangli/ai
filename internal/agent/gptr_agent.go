package agent

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal"
	"github.com/qiangli/ai/internal/docker/gptr"
)

type GptrAgent struct {
	config *internal.AppConfig

	Role   string
	Prompt string
}

func NewGptrAgent(cfg *internal.AppConfig) (*GptrAgent, error) {
	role := cfg.Role
	prompt := cfg.Prompt
	if role == "" {
		role = "system"
	}

	agent := GptrAgent{
		config: cfg,
		Role:   role,
		Prompt: prompt,
	}
	return &agent, nil
}

func (r *GptrAgent) Send(ctx context.Context, in *UserInput) (*ChatMessage, error) {
	// FIXME: This is a hack
	// better to config the base url and api key (and others) for gptr
	u, err := url.Parse(r.config.LLM.BaseUrl)
	if err != nil {
		return nil, err
	}

	if isLoopback(u.Host) {
		_, port, _ := net.SplitHostPort(u.Host)
		u.Host = "host.docker.internal:" + port
	}
	os.Setenv("OPENAI_API_BASE", u.String())
	os.Setenv("OPENAI_API_KEY", r.config.LLM.ApiKey)

	var content string

	if !r.config.LLM.DryRun {
		tempDir, err := os.MkdirTemp("", "gptr")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tempDir)

		err = gptr.GenerateReport(ctx, in.Input(), tempDir)
		if err != nil {
			return nil, err
		}

		content, err = readReport(tempDir)
		if err != nil {
			return nil, err
		}
	} else {
		content = r.config.LLM.DryRunContent
	}

	return &ChatMessage{
		Agent:   "GPTR",
		Content: content,
	}, nil
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
