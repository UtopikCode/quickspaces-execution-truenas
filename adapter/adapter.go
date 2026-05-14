package adapter

import (
	"encoding/json"
	"fmt"

	contracts "github.com/UtopikCode/quickspaces-execution-contracts"
)

type ExecutionAdapter = contracts.ExecutionAdapter

type DockerExecutionAdapter struct {
	client DockerClient
}

func NewDockerExecutionAdapter(client DockerClient) ExecutionAdapter {
	return &DockerExecutionAdapter{client: client}
}

func NewDefaultDockerExecutionAdapter() (contracts.ExecutionAdapter, error) {
	client, err := NewDockerClient()
	if err != nil {
		return nil, err
	}
	return NewDockerExecutionAdapter(client), nil
}

func NewDefaultAdapter() (contracts.ExecutionAdapter, error) {
	return NewDefaultDockerExecutionAdapter()
}

type dockerRuntimeConfig struct {
	Image string            `json:"image"`
	Env   map[string]string `json:"env"`
	Cmd   []string          `json:"cmd"`
	Ports map[string]string `json:"ports"`
}

var (
	ErrWorkspaceNotFound = &WorkspaceError{Code: "workspace_not_found", Message: "workspace not found"}
)

type WorkspaceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *WorkspaceError) Error() string {
	return e.Message
}

func IsWorkspaceNotFound(err error) bool {
	if err == nil {
		return false
	}
	if wErr, ok := err.(*WorkspaceError); ok {
		return wErr.Code == ErrWorkspaceNotFound.Code
	}
	return false
}

func parseDockerRuntimeConfig(profile contracts.ExecutionProfile) (dockerRuntimeConfig, error) {
	if len(profile.RuntimeConfig) == 0 {
		return dockerRuntimeConfig{}, fmt.Errorf("executionProfile.runtimeConfig must be supplied")
	}

	payload, err := json.Marshal(profile.RuntimeConfig)
	if err != nil {
		return dockerRuntimeConfig{}, fmt.Errorf("encode runtimeConfig: %w", err)
	}

	var cfg dockerRuntimeConfig
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return dockerRuntimeConfig{}, fmt.Errorf("parse docker runtimeConfig: %w", err)
	}

	if cfg.Image == "" {
		return dockerRuntimeConfig{}, fmt.Errorf("docker runtimeConfig missing image")
	}

	return cfg, nil
}

func containerNameFromWorkspace(ws contracts.Workspace) string {
	return ws.ID
}
