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
	"github.com/qiangli/ai/internal/api"
	"github.com/qiangli/ai/internal/docker/gptr"
	"github.com/qiangli/ai/internal/llm"
	"github.com/qiangli/ai/internal/log"
	"github.com/qiangli/ai/internal/resource"
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
	log.Debugf("GptrAgent.Send: subcommand: %s\n", in.Subcommand)
	if in.Subcommand == "" {
		return r.Handle(ctx, in, nil)
	}

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

	if !internal.DryRun {
		tempDir, err := os.MkdirTemp("", "gptr")
		if err != nil {
			return nil, err
		}
		defer os.RemoveAll(tempDir)

		err = gptr.GenerateReport(ctx, in.Subcommand, in.Input(), tempDir)
		if err != nil {
			return nil, err
		}

		content, err = readReport(tempDir)
		if err != nil {
			return nil, err
		}
	} else {
		content = internal.DryRunContent
	}

	return &ChatMessage{
		Agent:   "GPTR",
		Content: content,
	}, nil
}

func (r *GptrAgent) Handle(ctx context.Context, req *api.Request, next api.HandlerNext) (*ChatMessage, error) {
	var intent = req.Intent()
	if intent == "" {
		log.Debugf("GptrAgent.Handle: no intent, using default\n")
		req.Subcommand = "/research_report/objective"
		return r.Send(ctx, req)
	}

	prompt, err := resource.GetCliGptrReportSystem(gptr.ReportTypes, gptr.Tones)
	if err != nil {
		return nil, err
	}

	action := func(ctx context.Context, sub string) (string, error) {
		log.Debugf("action: GPTR subcommand: %s\n", sub)
		req.Subcommand = sub
		resp, err := r.Send(ctx, req)
		if err != nil {
			return "", err
		}
		return resp.Content, nil
	}

	model := internal.Level1(r.config.LLM)
	model.Tools = llm.GetGptrTools()

	msg := &internal.Message{
		Role:   "system",
		Prompt: prompt,
		Model:  model,
		Input:  intent,
		Next:   action,
	}

	resp, err := llm.Chat(ctx, msg)
	if err != nil {
		return nil, err
	}

	return &ChatMessage{
		Content: resp.Content,
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
