package adapter

import (
	"context"
	"fmt"
	"io"
	"strings"

	contracts "github.com/UtopikCode/quickspaces-execution-contracts"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/go-connections/nat"
)

func (d *DockerExecutionAdapter) StartWorkspace(ctx context.Context, ws contracts.Workspace) (contracts.WorkspaceState, error) {
	cfg, err := parseDockerRuntimeConfig(ws.ExecutionProfile)
	if err != nil {
		return contracts.WorkspaceStateError, err
	}

	if err := d.pullImage(ctx, cfg.Image); err != nil {
		return contracts.WorkspaceStateError, err
	}

	config := &container.Config{
		Image: cfg.Image,
		Env:   envList(cfg.Env),
		Cmd:   cfg.Cmd,
	}

	hostConfig := &container.HostConfig{}
	if len(cfg.Ports) > 0 {
		bindings, err := buildPortBindings(cfg.Ports)
		if err != nil {
			return contracts.WorkspaceStateError, err
		}
		hostConfig.PortBindings = bindings
	}

	resp, err := d.client.ContainerCreate(ctx, config, hostConfig, &network.NetworkingConfig{}, containerNameFromWorkspace(ws))
	if err != nil {
		return contracts.WorkspaceStateError, fmt.Errorf("create container: %w", err)
	}

	if err := d.client.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return contracts.WorkspaceStateError, fmt.Errorf("start container: %w", err)
	}

	return contracts.WorkspaceStateRunning, nil
}

func (d *DockerExecutionAdapter) StopWorkspace(ctx context.Context, id string) error {
	timeout := 10
	if err := d.client.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout}); err != nil {
		if cerrdefs.IsNotFound(err) {
			return ErrWorkspaceNotFound
		}
		return fmt.Errorf("stop container: %w", err)
	}
	return nil
}

func (d *DockerExecutionAdapter) GetWorkspaceStatus(ctx context.Context, id string) (contracts.WorkspaceState, error) {
	inspect, err := d.client.ContainerInspect(ctx, id)
	if err != nil {
		if cerrdefs.IsNotFound(err) {
			return contracts.WorkspaceStateError, ErrWorkspaceNotFound
		}
		return contracts.WorkspaceStateError, fmt.Errorf("inspect container: %w", err)
	}

	if inspect.State == nil {
		return contracts.WorkspaceStateError, fmt.Errorf("inspect container: missing state")
	}

	switch strings.ToLower(inspect.State.Status) {
	case "running":
		return contracts.WorkspaceStateRunning, nil
	case "created", "restarting":
		return contracts.WorkspaceStatePending, nil
	default:
		return contracts.WorkspaceStateStopped, nil
	}
}

func (d *DockerExecutionAdapter) pullImage(ctx context.Context, imageRef string) error {
	reader, err := d.client.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", imageRef, err)
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
