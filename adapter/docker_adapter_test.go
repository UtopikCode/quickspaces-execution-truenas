package adapter

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	contracts "github.com/UtopikCode/quickspaces-execution-contracts"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/errdefs"
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

func (m *mockDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return m.startErr
}

func (m *mockDockerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	return m.stopErr
}

func (m *mockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return m.inspectResult, m.inspectErr
}

func (m *mockDockerClient) ImagePull(ctx context.Context, ref string, options image.PullOptions) (io.ReadCloser, error) {
	m.pulledImages = append(m.pulledImages, ref)
	return io.NopCloser(strings.NewReader("")), nil
}

func TestDockerExecutionAdapter_StartWorkspace(t *testing.T) {
	mock := &mockDockerClient{
		createResponse: container.CreateResponse{ID: "container-1"},
	}
	adapter := NewDockerExecutionAdapter(mock)

	state, err := adapter.StartWorkspace(context.Background(), contracts.Workspace{
		ID: "workspace-1",
		ExecutionProfile: contracts.ExecutionProfile{
			RuntimeConfig: map[string]interface{}{
				"image": "alpine:latest",
				"env":   map[string]string{"FOO": "bar"},
				"ports": map[string]string{"8080/tcp": "8080"},
				"cmd":   []string{"sh", "-c", "echo hello"},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state != contracts.WorkspaceStateRunning {
		t.Fatalf("expected running state, got %v", state)
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
			ContainerJSONBase: &container.ContainerJSONBase{
				ID:    "container-1",
				Image: "alpine:latest",
				State: &types.ContainerState{Status: "running", Running: true, ExitCode: 0, StartedAt: "now"},
			},
			Config: &container.Config{Image: "alpine:latest"},
		},
	}
	adapter := NewDockerExecutionAdapter(mock)

	status, err := adapter.GetWorkspaceStatus(context.Background(), "workspace-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status != contracts.WorkspaceStateRunning {
		t.Fatalf("expected running state, got %v", status)
	}
}
