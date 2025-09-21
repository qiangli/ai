package oh

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	"github.com/docker/docker/api/types/mount"

	"github.com/qiangli/ai/internal/docker"
	"github.com/qiangli/ai/swarm/log"
)

// default
// WORKSPACE_MOUNT_PATH:WORKSPACE_MOUNT_PATH_IN_SANDBOX
const WorkspaceInSandbox = "/workspace"

//go:embed Dockerfile
var Dockerfile []byte

//go:embed cfg.env
var envFile string

const imageName = "ai/oh"
const containerName = "ai-oh-v0.19.0"

// BuildImage builds the OpenHands Docker image
func BuildImage(ctx context.Context) error {
	return docker.BuildDockerImage(ctx, "Dockerfile", imageName, Dockerfile)
}

func getEnvVarMap() map[string]string {
	vars := []string{
		"SANDBOX_RUNTIME_CONTAINER_IMAGE",
		"SANDBOX_USER_ID",
		"WORKSPACE_MOUNT_PATH",
		"WORKSPACE_MOUNT_PATH_IN_SANDBOX",
		"LLM_BASE_URL",
		"LLM_API_KEY",
		"LLM_MODEL",
		"LOG_ALL_EVENTS",
		"WORKSPACE_BASE",
	}
	envVars := docker.GetEnvVars(vars)

	// override env vars
	// workspace_base: Base path for the workspace. Defaults to `./workspace` as absolute path.
	// workspace_mount_path: Path to mount the workspace. Defaults to `workspace_base`.
	// workspace_mount_path_in_sandbox: Path to mount the workspace in sandbox. Defaults to `/workspace`.
	rteEnvVars := map[string]string{
		"SANDBOX_USER_ID": fmt.Sprintf("%v", os.Getuid()),
	}
	for key, value := range rteEnvVars {
		envVars[key] = value
	}

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

// https://docs.all-hands.dev/modules/usage/how-to/headless-mode
func RunContainer(ctx context.Context, query string) error {
	envVars := getEnvVarMap()
	args := []string{"python", "-m", "openhands.core.main", "-t", query}

	config := &docker.ContainerConfig{
		Image: imageName,
		Env:   toArray(envVars),
		Cmd:   args,
		User:  "root",
	}

	workspaceBase := envVars["WORKSPACE_BASE"]

	hostConfig := &docker.ContainerHostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: workspaceBase,
				Target: "/opt/workspace_base",
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
