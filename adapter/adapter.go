package adapter

import (
	contracts "github.com/UtopikCode/quickspaces-execution-contracts"
)

type ExecutionAdapter = contracts.ExecutionAdapter

var (
	ErrWorkspaceNotFound = &WorkspaceError{Code: "workspace_not_found", Message: "workspace not found"}
)

type WorkspaceError struct {
	Code    string
	Message string
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
