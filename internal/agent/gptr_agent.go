package agent

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/qiangli/ai/internal/gptr"
	"github.com/qiangli/ai/internal/llm"
)

type GptrAgent struct {
	config *llm.Config

	Role    string
	Message string
}

func NewGptrAgent(cfg *llm.Config, role, content string) (*GptrAgent, error) {
	if role == "" {
		role = "system"
	}

	agent := GptrAgent{
		config:  cfg,
		Role:    role,
		Message: content,
	}
	return &agent, nil
}

func isLoopback(hostport string) bool {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}

	ip := net.ParseIP(host)

	if ip != nil && ip.IsLoopback() {
		return true
	}

	if host == "localhost" {
		return true
	}

	return false
}

func (r *GptrAgent) Send(ctx context.Context, input string) (*ChatMessage, error) {
	// FIXME: This is a hack
	// better to config the base url and api key (and others) for gptr
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

	var content string

	if !r.config.DryRun {
		tempDir, err := os.MkdirTemp("", "gptr")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tempDir)

		err = gptr.GenerateReport(input, tempDir)
		if err != nil {
			return nil, err
		}

		content, err = readReport(tempDir)
		if err != nil {
			return nil, err
		}
	} else {
		content = r.config.DryRunContent
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
