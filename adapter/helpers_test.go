package adapter

import (
	"context"
	"errors"
	"testing"

	contracts "github.com/UtopikCode/quickspaces-execution-contracts"
	cerrdefs "github.com/containerd/errdefs"
	"github.com/docker/go-connections/nat"
)

func TestIsWorkspaceNotFound(t *testing.T) {
	if IsWorkspaceNotFound(nil) {
		t.Fatal("expected nil error to return false")
	}

	err := ErrWorkspaceNotFound
	if !IsWorkspaceNotFound(err) {
		t.Fatalf("expected workspace not found for %T", err)
	}

	if IsWorkspaceNotFound(errors.New("other")) {
		t.Fatal("expected non-workspace error to return false")
	}
}

func TestEnvList(t *testing.T) {
	if got := envList(nil); got != nil {
		t.Fatalf("expected nil env list for nil map, got %v", got)
	}

	got := envList(map[string]string{"FOO": "bar", "BAZ": "qux"})
	if len(got) != 2 {
		t.Fatalf("expected 2 env entries, got %d", len(got))
	}

	expected := map[string]struct{}{"FOO=bar": {}, "BAZ=qux": {}}
	for _, entry := range got {
		if _, ok := expected[entry]; !ok {
			t.Fatalf("unexpected env entry %q", entry)
		}
	}
}

func TestBuildPortBindings(t *testing.T) {
	bindings, err := buildPortBindings(map[string]string{"8080/tcp": "8080", "9090": "9090"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(bindings) != 2 {
		t.Fatalf("expected 2 port bindings, got %d", len(bindings))
	}

	if binding, ok := bindings[nat.Port("8080/tcp")]; !ok || len(binding) != 1 || binding[0].HostPort != "8080" {
		t.Fatalf("unexpected binding for 8080/tcp: %v", binding)
	}
	if binding, ok := bindings[nat.Port("9090/tcp")]; !ok || len(binding) != 1 || binding[0].HostPort != "9090" {
		t.Fatalf("unexpected binding for 9090/tcp: %v", binding)
	}
}

func TestBuildPortBindings_InvalidPort(t *testing.T) {
	_, err := buildPortBindings(map[string]string{"invalid": "8080"})
	if err == nil {
		t.Fatal("expected error for invalid port spec")
	}
}

func TestDockerExecutionAdapter_StopWorkspace_Error(t *testing.T) {
	mock := &mockDockerClient{stopErr: errors.New("timeout")}
	adapter := NewDockerExecutionAdapter(mock)

	err := adapter.StopWorkspace(context.Background(), "workspace-1")
	if err == nil {
		t.Fatal("expected an error when stop fails")
	}
	if IsWorkspaceNotFound(err) {
		t.Fatalf("expected a wrapped error, not workspace not found, got %v", err)
	}
}

func TestDockerExecutionAdapter_GetWorkspaceStatus_NotFound(t *testing.T) {
	mock := &mockDockerClient{inspectErr: cerrdefs.ErrNotFound.WithMessage("not found")}
	adapter := NewDockerExecutionAdapter(mock)

	status, err := adapter.GetWorkspaceStatus(context.Background(), "workspace-unknown")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !IsWorkspaceNotFound(err) {
		t.Fatalf("expected workspace not found error, got %v", err)
	}
	if status != contracts.WorkspaceStateError {
		t.Fatalf("expected error workspace state, got %v", status)
	}
}
