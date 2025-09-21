package docker

import (
	"archive/tar"
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/qiangli/ai/swarm/log"
)

type ContainerConfig = container.Config
type ContainerHostConfig = container.HostConfig

func ParseEnvFile(envData string) map[string]string {
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

func GetEnvVars(vars []string) map[string]string {
	envMap := make(map[string]string)

	for _, key := range vars {
		v := strings.TrimSpace(os.Getenv(key))
		if v != "" {
			envMap[key] = v
		}
	}

	return envMap
}

// BuildDockerImage constructs and builds a Docker image using provided contents and parameters.
func BuildDockerImage(ctx context.Context, dockerfileName, tag string, dockerfileContent []byte) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	tarBuffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(tarBuffer)

	// Add Dockerfile to the tar buffer
	if err := AddFileToTar(tarWriter, dockerfileName, dockerfileContent); err != nil {
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
	if log.IsNormal() {
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

// AddFileToTar add a file to the tarball
func AddFileToTar(tw *tar.Writer, name string, fileContent []byte) error {
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

func RunContainer(ctx context.Context, containerName string, config *container.Config, hostConfig *container.HostConfig) (*container.CreateResponse, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, containerName)
	// try remove and create again if container already exists
	if errdefs.IsConflict(err) {
		RemoveContainer(ctx, containerName)

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

func RemoveContainer(ctx context.Context, containerName string) error {
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

// GetCurrentUG returns the current user's uid:gid.
// It uses command execution as a fallback if the Go user package fails.
func GetCurrentUG() string {
	currentUser, err := user.Current()
	if err == nil {
		return fmt.Sprintf("%s:%s", currentUser.Uid, currentUser.Gid)
	}

	uid, err := execCommand("id", "-u")
	if err != nil {
		uid = "0"
	}
	gid, err := execCommand("id", "-g")
	if err != nil {
		gid = "0"
	}
	return fmt.Sprintf("%s:%s", uid, gid)
}

func execCommand(name string, arg string) (string, error) {
	cmd := exec.Command(name, arg)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}
