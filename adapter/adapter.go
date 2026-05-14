package adapter

import "context"

type ExecutionAdapter interface {
	StartWorkspace(ctx context.Context, workspaceID string, image string, options WorkspaceOptions) (string, error)
	StopWorkspace(ctx context.Context, workspaceID string) error
	GetWorkspaceStatus(ctx context.Context, workspaceID string) (WorkspaceStatus, error)
}

type WorkspaceOptions struct {
	Env   map[string]string
	Ports map[string]string
	Cmd   []string
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
