package gptr

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/qiangli/ai/internal/docker"
	"github.com/qiangli/ai/swarm/log"
)

//go:embed Dockerfile
var Dockerfile []byte

//go:embed cfg.env
var envFile string

const imageName = "ai/gptr"
const containerName = "ai-gptr-v3.1.7"

// BuildImage builds the GPT Researcher Docker image
func BuildImage(ctx context.Context) error {
	return docker.BuildDockerImage(ctx, "Dockerfile", imageName, Dockerfile)
}

func getEnvVars() []string {
	vars := []string{
		"OPENAI_API_BASE",
		"OPENAI_API_KEY",
		"RETRIEVER",
		"GOOGLE_API_KEY",
		"RETRIEVER_ARG_API_KEY",
		"SEARX_URL",
		"RETRIEVER_ENDPOINT",
		"EMBEDDING",
		"FAST_LLM",
		"SMART_LLM",
		"STRATEGIC_LLM",
		"CURATE_SOURCES",
		"REPORT_FORMAT",
		"DOC_PATH",
		"SCRAPER",
	}
	//
	defaultEnvVars := docker.ParseEnvFile(envFile)
	envVars := docker.GetEnvVars(vars)
	for key, value := range defaultEnvVars {
		if _, exists := envVars[key]; !exists {
			envVars[key] = value
		}
	}

	var envVarsSlice []string
	for key, value := range envVars {
		envVarsSlice = append(envVarsSlice, fmt.Sprintf("%s=%s", key, value))
	}
	return envVarsSlice
}

type GptrQuery struct {
	Query      string `json:"query"`
	ReportType string `json:"report_type"`
	Tone       string `json:"tone"`
}

func RunContainer(ctx context.Context, reportType, tone, query string, outDir string) error {
	envVars := getEnvVars()

	output, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}

	args := []string{query, "--report_type", reportType}
	config := &docker.ContainerConfig{
		Image: imageName,
		Env:   envVars,
		Cmd:   args,
	}

	log.Debugf("config: %+v\n", config)

	hostConfig := &docker.ContainerHostConfig{
		Binds: []string{output + ":/app/outputs/"},
	}

	log.Debugf("hostConfig: %+v\n", hostConfig)

	_, err = docker.RunContainer(ctx, containerName, config, hostConfig)
	if err != nil {
		log.Errorf("Error running container: %v\n", err)

		// Attempt to remove the container
		if rmErr := docker.RemoveContainer(ctx, containerName); rmErr != nil {
			log.Errorf("Error removing container: %v\n", rmErr)
		}
		return err
	}

	return nil
}
