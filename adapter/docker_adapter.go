package adapter

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/errdefs"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

type DockerExecutionAdapter struct {
	client DockerClient
}

func NewDockerExecutionAdapter(client DockerClient) *DockerExecutionAdapter {
	return &DockerExecutionAdapter{client: client}
}

func (d *DockerExecutionAdapter) StartWorkspace(ctx context.Context, workspaceID string, image string, options WorkspaceOptions) (string, error) {
	if err := d.pullImage(ctx, image); err != nil {
		return "", err
	}

	config := &container.Config{
		Image: image,
		Env:   envList(options.Env),
		Cmd:   options.Cmd,
	}

	hostConfig := &container.HostConfig{}
	if len(options.Ports) > 0 {
		bindings, err := buildPortBindings(options.Ports)
		if err != nil {
			return "", err
		}
		hostConfig.PortBindings = bindings
	}

	resp, err := d.client.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, workspaceID)
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}

	if err := d.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return "", fmt.Errorf("start container: %w", err)
	}

	return resp.ID, nil
}

func (d *DockerExecutionAdapter) StopWorkspace(ctx context.Context, workspaceID string) error {
	timeout := 10 * time.Second
	if err := d.client.ContainerStop(ctx, workspaceID, &timeout); err != nil {
		if errdefs.IsNotFound(err) {
			return ErrWorkspaceNotFound
		}
		return fmt.Errorf("stop container: %w", err)
	}
	return nil
}

func (d *DockerExecutionAdapter) GetWorkspaceStatus(ctx context.Context, workspaceID string) (WorkspaceStatus, error) {
	inspect, err := d.client.ContainerInspect(ctx, workspaceID)
	if err != nil {
		if errdefs.IsNotFound(err) {
			return WorkspaceStatus{State: "NotFound"}, ErrWorkspaceNotFound
		}
		return WorkspaceStatus{}, fmt.Errorf("inspect container: %w", err)
	}

	status := WorkspaceStatus{
		ContainerID: inspect.ID,
		Image:       inspect.Config.Image,
		State:       inspect.State.Status,
		Status:      inspect.State.Status,
		Running:     inspect.State.Running,
		ExitCode:    inspect.State.ExitCode,
		StartedAt:   inspect.State.StartedAt,
		FinishedAt:  inspect.State.FinishedAt,
	}

	return status, nil
}

func (d *DockerExecutionAdapter) pullImage(ctx context.Context, image string) error {
	reader, err := d.client.ImagePull(ctx, image, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", image, err)
	}
	defer reader.Close()
	_, _ = io.Copy(io.Discard, reader)
	return nil
}

func envList(env map[string]string) []string {
	if len(env) == 0 {
		return nil
	}

	list := make([]string, 0, len(env))
	for key, value := range env {
		list = append(list, fmt.Sprintf("%s=%s", key, value))
	}
	return list
}

func buildPortBindings(ports map[string]string) (nat.PortMap, error) {
	bindings := nat.PortMap{}
	for containerPort, hostPort := range ports {
		protocol := "tcp"
		portSpec := containerPort
		if strings.Contains(containerPort, "/") {
			parts := strings.SplitN(containerPort, "/", 2)
			portSpec = parts[0]
			protocol = parts[1]
		}
		port, err := nat.NewPort(protocol, portSpec)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q: %w", containerPort, err)
		}
		bindings[port] = []nat.PortBinding{{HostPort: hostPort}}
	}
	return bindings, nil
}
