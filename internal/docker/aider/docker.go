package aider

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/mount"

	"github.com/qiangli/ai/internal/docker"
	"github.com/qiangli/ai/internal/log"
)

const WorkspaceInSandbox = "/app"

type ChatMode string

const (
	Code      ChatMode = "code"
	Ask       ChatMode = "ask"
	Architect ChatMode = "architect"
)

//go:embed Dockerfile
var Dockerfile []byte

//go:embed cfg.env
var envFile string

const imageName = "ai/aider"
const containerName = "ai-aider-v0.71.0"

// BuildImage builds the OpenHands Docker image
func BuildImage(ctx context.Context) error {
	return docker.BuildDockerImage(ctx, "Dockerfile", imageName, Dockerfile)
}

func getEnvVars() []string {
	vars := []string{
		"WORKSPACE_BASE",
	}
	envVars := docker.GetEnvVars(vars)

	// add default env vars if not set
	defaultEnvVars := docker.ParseEnvFile(envFile)
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

func RunContainer(ctx context.Context, mode ChatMode, input string) error {
	envVars := getEnvVars()
	args := []string{"--chat-mode", string(mode), "--message", input}

	config := &docker.ContainerConfig{
		Image: imageName,
		Env:   envVars,
		Cmd:   args,
		User:  docker.GetCurrentUG(),
	}

	workspaceBase := os.Getenv("WORKSPACE_BASE")

	hostConfig := &docker.ContainerHostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: workspaceBase,
				Target: WorkspaceInSandbox,
			},
			{
				Type:   mount.TypeBind,
				Source: "/var/run/docker.sock",
				Target: "/var/run/docker.sock",
			},
		},
		ExtraHosts: []string{"host.docker.internal:host-gateway"},
	}

	log.Debugf("hostConfig: %+v\n", hostConfig)

	_, err := docker.RunContainer(ctx, containerName, config, hostConfig)
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
