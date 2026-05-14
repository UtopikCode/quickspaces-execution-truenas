package adapter

type WorkspaceStatus struct {
	ContainerID string `json:"container_id,omitempty"`
	Image       string `json:"image,omitempty"`
	State       string `json:"state,omitempty"`
	Status      string `json:"status,omitempty"`
	Running     bool   `json:"running,omitempty"`
	ExitCode    int    `json:"exit_code,omitempty"`
	StartedAt   string `json:"started_at,omitempty"`
	FinishedAt  string `json:"finished_at,omitempty"`
}
