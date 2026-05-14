package adapter

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/errdefs"
	"github.com/docker/docker/api/types/network"
)

type mockDockerClient struct {
	createResponse container.CreateResponse
	createErr      error
	startErr       error
	stopErr        error
	inspectResult  types.ContainerJSON
	inspectErr     error
	pulledImages   []string
}

func (m *mockDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.CreateResponse, error) {
	return m.createResponse, m.createErr
}

func (m *mockDockerClient) ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	return m.startErr
}

func (m *mockDockerClient) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	return m.stopErr
}

func (m *mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return m.inspectResult, m.inspectErr
}

func (m *mockDockerClient) ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error) {
	m.pulledImages = append(m.pulledImages, ref)
	return io.NopCloser(strings.NewReader("")), nil
}

func TestDockerExecutionAdapter_StartWorkspace(t *testing.T) {
	mock := &mockDockerClient{
		createResponse: container.CreateResponse{ID: "container-1"},
	}
	adapter := NewDockerExecutionAdapter(mock)

	id, err := adapter.StartWorkspace(context.Background(), "workspace-1", "alpine:latest", WorkspaceOptions{
		Env:   map[string]string{"FOO": "bar"},
		Ports: map[string]string{"8080/tcp": "8080"},
		Cmd:   []string{"sh", "-c", "echo hello"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id != "container-1" {
		t.Fatalf("expected container ID %q, got %q", "container-1", id)
	}
	if len(mock.pulledImages) != 1 || mock.pulledImages[0] != "alpine:latest" {
		t.Fatalf("expected image to be pulled, got %v", mock.pulledImages)
	}
}

func TestDockerExecutionAdapter_StopWorkspace_NotFound(t *testing.T) {
	mock := &mockDockerClient{stopErr: errdefs.NotFound(fmt.Errorf("no such container"))}
	adapter := NewDockerExecutionAdapter(mock)

	err := adapter.StopWorkspace(context.Background(), "workspace-unknown")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !IsWorkspaceNotFound(err) {
		t.Fatalf("expected workspace not found error, got %v", err)
	}
}

func TestDockerExecutionAdapter_GetWorkspaceStatus(t *testing.T) {
	mock := &mockDockerClient{
		inspectResult: types.ContainerJSON{
			ID:     "container-1",
			Config: &container.Config{Image: "alpine:latest"},
			State:  &types.ContainerState{Status: "running", Running: true, ExitCode: 0, StartedAt: "now"},
		},
	}
	adapter := NewDockerExecutionAdapter(mock)

	status, err := adapter.GetWorkspaceStatus(context.Background(), "workspace-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status.State != "running" || !status.Running {
		t.Fatalf("expected running state, got %#v", status)
	}
}
