package gptr

import (
	"archive/tar"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/qiangli/ai/internal/log"
)

//go:embed dockerfile/gptr.Dockerfile
var gptrDockerfile []byte

//go:embed dockerfile/gptr.env
var gptrEnvFile string

const gptrImageName = "ai/gptr"
const gptrContainerName = "ai-gptr-v3.1.7"

func parseEnvFile(envData string) map[string]string {
	lines := strings.Split(envData, "\n")
	envVars := make(map[string]string)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Split the line into key and value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			envVars[key] = value
		}
	}

	return envVars
}

func getEnvVars() map[string]string {
	envVars := []string{
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

	envMap := make(map[string]string)

	for _, key := range envVars {
		v := strings.TrimSpace(os.Getenv(key))
		if v != "" {
			envMap[key] = v
		}
	}

	return envMap
}

// buildDockerImage constructs and builds a Docker image using provided contents and parameters.
func buildDockerImage(ctx context.Context, dockerfileName, tag string, dockerfileContent []byte) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	tarBuffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(tarBuffer)

	// Add Dockerfile to the tar buffer
	if err := addFileToTar(tarWriter, dockerfileName, dockerfileContent); err != nil {
		return err
	}

	if err := tarWriter.Close(); err != nil {
		return err
	}

	// Prepare to build the image
	tarReader := bytes.NewReader(tarBuffer.Bytes())
	buildOptions := types.ImageBuildOptions{
		Context:        tarReader,
		Dockerfile:     dockerfileName,
		Tags:           []string{tag},
		Remove:         true,
		SuppressOutput: false,
		PullParent:     true,
	}

	resp, err := cli.ImageBuild(ctx, tarReader, buildOptions)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var out io.Writer
	if log.IsVerbose() {
		out = os.Stderr
	} else {
		out = io.Discard
	}

	err = jsonmessage.DisplayJSONMessagesStream(resp.Body, out, os.Stdout.Fd(), true, nil)
	if err != nil {
		return err
	}
	log.Infof("Image build with tag %s succeeded\n", tag)

	return nil
}

// Utility function to add a file to the tarball
func addFileToTar(tw *tar.Writer, name string, fileContent []byte) error {
	header := &tar.Header{
		Name: name,
		Mode: 0600,
		Size: int64(len(fileContent)),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	if _, err := tw.Write(fileContent); err != nil {
		return err
	}
	return nil
}

// BuildGPTRImage builds the GPT Researcher Docker image
func BuildGPTRImage(ctx context.Context) error {
	return buildDockerImage(ctx, "gptr.Dockerfile", gptrImageName, gptrDockerfile)
}

func getEnvFile() []string {
	defaultEnvVars := parseEnvFile(gptrEnvFile)
	envVars := getEnvVars()
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

func RunGPTRContainer(ctx context.Context, query string, outDir string) error {
	envVars := getEnvFile()

	output, err := filepath.Abs(outDir)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}

	args := []string{query, "--report_type", "research_report"}
	config := &container.Config{
		Image: gptrImageName,
		Env:   envVars,
		Cmd:   args,
	}

	log.Debugf("config: %+v\n", config)

	hostConfig := &container.HostConfig{
		Binds: []string{output + ":/app/outputs/"},
	}

	log.Debugf("hostConfig: %+v\n", hostConfig)

	_, err = runContainer(ctx, gptrContainerName, config, hostConfig)
	if err != nil {
		log.Errorf("Error running container: %v\n", err)

		// Attempt to remove the container
		if rmErr := removeContainer(ctx, gptrContainerName); rmErr != nil {
			log.Errorf("Error removing container: %v\n", rmErr)
		}
		return err
	}

	return nil
}

func runContainer(ctx context.Context, containerName string, config *container.Config, hostConfig *container.HostConfig) (*container.CreateResponse, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	// try remove and create again if container already exists
	if errdefs.IsConflict(err) {
		removeContainer(ctx, containerName)

		resp, err = cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
		if err != nil {
			return nil, err
		}
	}
	if err != nil {
		return nil, err
	}

	var attachOptions container.AttachOptions

	if log.IsQuiet() {
		attachOptions.Stream = false
		attachOptions.Stdout = false
		attachOptions.Stderr = false
	} else {
		attachOptions.Stream = true
		attachOptions.Stdout = true
		attachOptions.Stderr = true
	}

	hjResp, err := cli.ContainerAttach(ctx, resp.ID, attachOptions)
	if err != nil {
		return &resp, err
	}
	defer hjResp.Close()

	// progress output
	go func() {
		if _, err := stdcopy.StdCopy(os.Stderr, os.Stderr, hjResp.Reader); err != nil {
			log.Errorf("error copying output: %v\n", err)
		}
	}()

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return &resp, err
	}

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)

	select {
	case err := <-errCh:
		if err != nil {
			return &resp, err
		}
	case <-statusCh:
	}

	return &resp, nil
}

func removeContainer(ctx context.Context, containerName string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	// Get the container ID from the container name
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return err
	}

	var containerID string
	for _, container := range containers {
		for _, name := range container.Names {
			if name == "/"+containerName {
				containerID = container.ID
				break
			}
		}
		if containerID != "" {
			break
		}
	}

	if containerID == "" {
		log.Errorf("Container %s not found\n", containerName)
		return nil
	}

	// Remove the container
	if err := cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return err
	}

	log.Debugf("Container %s removed successfully\n", containerName)
	return nil
}
