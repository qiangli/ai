package aider

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/docker/docker/api/types/mount"

	"github.com/qiangli/ai/internal/docker"
	"github.com/qiangli/ai/swarm/log"
)

const WorkspaceInSandbox = "/app"

type ChatMode string

const (
	Code      ChatMode = "code"
	Ask       ChatMode = "ask"
	Architect ChatMode = "architect"
	Help      ChatMode = "help"
	//
	Watch ChatMode = "watch"
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

func getEnvVarMap() map[string]string {
	vars := []string{
		"WORKSPACE_BASE",
		"OPENAI_API_BASE",
		"OPENAI_API_KEY",
		"AIDER_WEAK_MODEL",
		"AIDER_EDITOR_MODEL",
		"AIDER_MODEL",
		"AIDER_VERBOSE",
	}
	envVars := docker.GetEnvVars(vars)

	// add default env vars if not set
	defaultEnvVars := docker.ParseEnvFile(envFile)
	for key, value := range defaultEnvVars {
		if _, exists := envVars[key]; !exists {
			envVars[key] = value
		}
	}
	return envVars
}

func toArray(envVars map[string]string) []string {
	var envVarsSlice []string
	for key, value := range envVars {
		envVarsSlice = append(envVarsSlice, fmt.Sprintf("%s=%s", key, value))
	}
	return envVarsSlice
}

func RunContainer(ctx context.Context, mode ChatMode, input string) error {
	envVars := getEnvVarMap()
	var args []string
	if mode == Watch {
		args = []string{"--watch-files", "--message", input}
	} else {
		args = []string{"--chat-mode", string(mode), "--message", input}
	}

	config := &docker.ContainerConfig{
		Image: imageName,
		Env:   toArray(envVars),
		Cmd:   args,
		User:  docker.GetCurrentUG(),
	}

	workspaceBase := envVars["WORKSPACE_BASE"]

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

	log.GetLogger(ctx).Debugf("hostConfig: %+v\n", hostConfig)

	_, err := docker.RunContainer(ctx, containerName, config, hostConfig)
	if err != nil {
		log.GetLogger(ctx).Errorf("Error running container: %v\n", err)

		// Attempt to remove the container
		if rmErr := docker.RemoveContainer(ctx, containerName); rmErr != nil {
			log.GetLogger(ctx).Errorf("Error removing container: %v\n", rmErr)
		}
		return err
	}

	return nil
}
