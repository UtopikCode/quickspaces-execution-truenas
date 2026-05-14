package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	contracts "github.com/UtopikCode/quickspaces-execution-contracts"
)

type mockScaleClient struct {
	createdPayload  scaleVMCreateRequest
	createdResponse *scaleVMResponse
	createdErr      error
	startErr        error
	shutdownErr     error
	statusResponse  string
	statusErr       error
	validateErr     error
}

func (m *mockScaleClient) CreateVM(ctx context.Context, payload scaleVMCreateRequest) (*scaleVMResponse, error) {
	m.createdPayload = payload
	return m.createdResponse, m.createdErr
}

func (m *mockScaleClient) StartVM(ctx context.Context, vmID string) error {
	return m.startErr
}

func (m *mockScaleClient) ShutdownVM(ctx context.Context, vmID string) error {
	return m.shutdownErr
}

func (m *mockScaleClient) GetVMStatus(ctx context.Context, vmID string) (string, error) {
	return m.statusResponse, m.statusErr
}

func (m *mockScaleClient) Validate(ctx context.Context) error {
	return m.validateErr
}

func TestScaleExecutionAdapter_StartWorkspace(t *testing.T) {
	mock := &mockScaleClient{
		createdResponse: &scaleVMResponse{ID: 42, Status: "created"},
	}
	adapter := newScaleExecutionAdapter(mock)

	state, err := adapter.StartWorkspace(context.Background(), contracts.Workspace{
		ID:    "ws-1",
		Owner: "alice",
		Repo:  "example/repo",
		ExecutionProfile: contracts.ExecutionProfile{
			RuntimeConfig: map[string]interface{}{
				"image":             "ubuntu-22.04-cloudimg-amd64.qcow2",
				"sshAuthorizedKeys": []string{"ssh-rsa AAA... user@example.com"},
			},
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state != contracts.WorkspaceStateRunning {
		t.Fatalf("expected running state, got %v", state)
	}
	if mock.createdPayload.Name != "ws-1" {
		t.Fatalf("expected vm name ws-1, got %q", mock.createdPayload.Name)
	}
	if len(mock.createdPayload.Devices) < 2 {
		t.Fatalf("expected at least disk and nic devices, got %d", len(mock.createdPayload.Devices))
	}
}

func TestScaleExecutionAdapter_StopWorkspace_NotFound(t *testing.T) {
	mock := &mockScaleClient{shutdownErr: ErrWorkspaceNotFound}
	adapter := newScaleExecutionAdapter(mock)

	err := adapter.StopWorkspace(context.Background(), "missing-vm")
	if err == nil {
		t.Fatal("expected error for missing workspace")
	}
	if !IsWorkspaceNotFound(err) {
		t.Fatalf("expected workspace not found error, got %v", err)
	}
}

func TestScaleExecutionAdapter_GetWorkspaceStatus(t *testing.T) {
	mock := &mockScaleClient{statusResponse: "running"}
	adapter := newScaleExecutionAdapter(mock)

	status, err := adapter.GetWorkspaceStatus(context.Background(), "vm-123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status != contracts.WorkspaceStateRunning {
		t.Fatalf("expected running state, got %v", status)
	}
}

func TestNewScaleExecutionAdapterFromHostConfig_ParsesHostConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/system/info" {
			t.Fatalf("expected system info endpoint, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hostname":"truenas"}`))
	}))
	defer srv.Close()

	hostConfig := json.RawMessage(fmt.Sprintf(`{"host":"%s","apiToken":"token123","insecure":true}`, srv.URL))
	adapter, err := NewScaleExecutionAdapterFromHostConfig(hostConfig)
	if err != nil {
		t.Fatalf("expected no error creating scale adapter from host config, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected adapter, got nil")
	}
}

func TestNewScaleExecutionAdapterFromHostConfig_InvalidHostConfig(t *testing.T) {
	hostConfig := json.RawMessage(`{"host":"http://bad-url%%%"}`)
	_, err := NewScaleExecutionAdapterFromHostConfig(hostConfig)
	if err == nil {
		t.Fatal("expected error for invalid host config")
	}
}

func TestNewScaleExecutionAdapterFromHostConfig_HostOffline(t *testing.T) {
	hostConfig := json.RawMessage(`{"host":"http://127.0.0.1:1","apiToken":"token123"}`)
	_, err := NewScaleExecutionAdapterFromHostConfig(hostConfig)
	if err == nil {
		t.Fatal("expected error when host is unreachable")
	}
}

func TestNewScaleExecutionAdapterFromHostConfig_ValidatesHostOnline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2.0/system/info" {
			t.Fatalf("expected system info endpoint, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hostname":"truenas"}`))
	}))
	defer srv.Close()

	hostConfig := json.RawMessage(fmt.Sprintf(`{"host":"%s","apiToken":"token123"}`, srv.URL))
	adapter, err := NewScaleExecutionAdapterFromHostConfig(hostConfig)
	if err != nil {
		t.Fatalf("expected no error creating scale adapter from host config, got %v", err)
	}
	if adapter == nil {
		t.Fatal("expected adapter, got nil")
	}
}

func TestNewScaleExecutionAdapterFromHostConfig_MissingHost(t *testing.T) {
	hostConfig := json.RawMessage(`{"apiToken":"token123"}`)
	_, err := NewScaleExecutionAdapterFromHostConfig(hostConfig)
	if err == nil {
		t.Fatal("expected error when host is missing")
	}
}
