package adapter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	contracts "github.com/UtopikCode/quickspaces-execution-contracts"
)

type ScaleExecutionAdapter struct {
	client ScaleClient
}

func NewScaleExecutionAdapterFromHostConfig(hostConfig json.RawMessage) (contracts.ExecutionAdapter, error) {
	cfg, err := parseScaleHostConfig(hostConfig)
	if err != nil {
		return nil, err
	}
	client, err := newScaleClient(cfg)
	if err != nil {
		return nil, err
	}
	return newScaleExecutionAdapter(client), nil
}

func newScaleExecutionAdapter(client ScaleClient) ExecutionAdapter {
	return &ScaleExecutionAdapter{client: client}
}

type scaleHostConfig struct {
	Host     string `json:"host,omitempty"`
	ApiToken string `json:"apiToken,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Insecure bool   `json:"insecure,omitempty"`
}

func parseScaleHostConfig(hostConfig json.RawMessage) (scaleHostConfig, error) {
	if len(hostConfig) == 0 {
		return scaleHostConfig{}, nil
	}

	var cfg scaleHostConfig
	if err := json.Unmarshal(hostConfig, &cfg); err != nil {
		return cfg, fmt.Errorf("parse truenas host config: %w", err)
	}

	return cfg, nil
}

type scaleProfile struct {
	VCPUs    int
	MemoryMB int
	DiskGB   int
	Bridge   string
}

var scaleProfiles = map[string]scaleProfile{
	"small":  {VCPUs: 2, MemoryMB: 4096, DiskGB: 20, Bridge: "bridge0"},
	"medium": {VCPUs: 4, MemoryMB: 8192, DiskGB: 40, Bridge: "bridge0"},
	"large":  {VCPUs: 8, MemoryMB: 16384, DiskGB: 80, Bridge: "bridge0"},
}

type scaleRuntimeConfig struct {
	Name              string   `json:"name,omitempty"`
	Profile           string   `json:"profile,omitempty"`
	Image             string   `json:"image"`
	VCPUs             int      `json:"vcpus,omitempty"`
	MemoryMB          int      `json:"memoryMB,omitempty"`
	DiskGB            int      `json:"diskGB,omitempty"`
	Bridge            string   `json:"bridge,omitempty"`
	SSHAuthorizedKeys []string `json:"sshAuthorizedKeys,omitempty"`
	UserData          string   `json:"userData,omitempty"`
}

func parseScaleRuntimeConfig(profile contracts.ExecutionProfile) (scaleRuntimeConfig, error) {
	if len(profile.RuntimeConfig) == 0 {
		return scaleRuntimeConfig{}, fmt.Errorf("executionProfile.runtimeConfig must be supplied")
	}

	payload, err := json.Marshal(profile.RuntimeConfig)
	if err != nil {
		return scaleRuntimeConfig{}, fmt.Errorf("encode runtimeConfig: %w", err)
	}

	var cfg scaleRuntimeConfig
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return scaleRuntimeConfig{}, fmt.Errorf("parse scale runtimeConfig: %w", err)
	}

	if cfg.Image == "" {
		return scaleRuntimeConfig{}, fmt.Errorf("scale runtimeConfig missing image")
	}

	if cfg.Profile != "" {
		profileName := strings.ToLower(strings.TrimSpace(cfg.Profile))
		if defaults, ok := scaleProfiles[profileName]; ok {
			if cfg.VCPUs == 0 {
				cfg.VCPUs = defaults.VCPUs
			}
			if cfg.MemoryMB == 0 {
				cfg.MemoryMB = defaults.MemoryMB
			}
			if cfg.DiskGB == 0 {
				cfg.DiskGB = defaults.DiskGB
			}
			if cfg.Bridge == "" {
				cfg.Bridge = defaults.Bridge
			}
		} else {
			return cfg, fmt.Errorf("unknown vm profile %q", cfg.Profile)
		}
	}

	if cfg.VCPUs <= 0 {
		cfg.VCPUs = 2
	}
	if cfg.MemoryMB <= 0 {
		cfg.MemoryMB = 2048
	}
	if cfg.DiskGB <= 0 {
		cfg.DiskGB = 20
	}
	if cfg.Bridge == "" {
		cfg.Bridge = "bridge0"
	}

	return cfg, nil
}

func (s *ScaleExecutionAdapter) StartWorkspace(ctx context.Context, ws contracts.Workspace) (contracts.WorkspaceState, error) {
	cfg, err := parseScaleRuntimeConfig(ws.ExecutionProfile)
	if err != nil {
		return contracts.WorkspaceStateError, err
	}

	if cfg.Name == "" {
		cfg.Name = ws.ID
	}

	vm, err := s.client.CreateVM(ctx, newScaleVMCreateRequest(ws, cfg))
	if err != nil {
		return contracts.WorkspaceStateError, fmt.Errorf("create scale vm: %w", err)
	}

	if err := s.client.StartVM(ctx, fmt.Sprintf("%d", vm.ID)); err != nil {
		return contracts.WorkspaceStateError, fmt.Errorf("start scale vm: %w", err)
	}

	return contracts.WorkspaceStateRunning, nil
}

func (s *ScaleExecutionAdapter) StopWorkspace(ctx context.Context, id string) error {
	if err := s.client.ShutdownVM(ctx, id); err != nil {
		if IsWorkspaceNotFound(err) {
			return ErrWorkspaceNotFound
		}
		return fmt.Errorf("shutdown scale vm: %w", err)
	}
	return nil
}

func (s *ScaleExecutionAdapter) GetWorkspaceStatus(ctx context.Context, id string) (contracts.WorkspaceState, error) {
	status, err := s.client.GetVMStatus(ctx, id)
	if err != nil {
		if IsWorkspaceNotFound(err) {
			return contracts.WorkspaceStateError, ErrWorkspaceNotFound
		}
		return contracts.WorkspaceStateError, fmt.Errorf("get scale vm status: %w", err)
	}

	switch strings.ToLower(status) {
	case "running", "started", "active":
		return contracts.WorkspaceStateRunning, nil
	case "starting", "pending", "configuring", "booting":
		return contracts.WorkspaceStatePending, nil
	case "stopped", "halted", "shutdown", "shut off":
		return contracts.WorkspaceStateStopped, nil
	default:
		return contracts.WorkspaceStateError, nil
	}
}

type ScaleClient interface {
	CreateVM(ctx context.Context, payload scaleVMCreateRequest) (*scaleVMResponse, error)
	StartVM(ctx context.Context, vmID string) error
	ShutdownVM(ctx context.Context, vmID string) error
	GetVMStatus(ctx context.Context, vmID string) (string, error)
}

type ScaleClientImpl struct {
	baseURL *url.URL
	client  *http.Client
	auth    scaleAuth
}

type scaleAuth struct {
	token    string
	username string
	password string
}

func newScaleClient(cfg scaleHostConfig) (ScaleClient, error) {
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "http://127.0.0.1"
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}

	baseURL, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("parse scale host: %w", err)
	}
	baseURL.Path = strings.TrimRight(baseURL.Path, "/")

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if cfg.Insecure {
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{}
		}
		transport.TLSClientConfig.InsecureSkipVerify = true
	}

	return &ScaleClientImpl{
		baseURL: baseURL,
		client:  &http.Client{Transport: transport, Timeout: 60 * time.Second},
		auth:    scaleAuth{token: cfg.ApiToken, username: cfg.Username, password: cfg.Password},
	}, nil
}

func (s *ScaleClientImpl) endpoint(relativePath string) string {
	rel := strings.TrimPrefix(relativePath, "/")
	apiBase := s.baseURL.Path
	if !strings.HasSuffix(apiBase, "/api/v2.0") {
		apiBase = path.Join(apiBase, "api/v2.0")
	}

	resolved := *s.baseURL
	resolved.Path = path.Join(apiBase, rel)
	return resolved.String()
}

func (s *ScaleClientImpl) doJSON(ctx context.Context, method, endpoint string, request any, response any) error {
	var body io.Reader
	if request != nil {
		payload, err := json.Marshal(request)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		body = strings.NewReader(string(payload))
	}

	req, err := http.NewRequestWithContext(ctx, method, s.endpoint(endpoint), body)
	if err != nil {
		return err
	}
	if request != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.auth.token != "" {
		req.Header.Set("Authorization", "Bearer "+s.auth.token)
	} else if s.auth.username != "" || s.auth.password != "" {
		req.SetBasicAuth(s.auth.username, s.auth.password)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrWorkspaceNotFound
	}
	if resp.StatusCode >= 300 {
		payload, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("scale api %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if response != nil {
		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
	return nil
}

func (s *ScaleClientImpl) CreateVM(ctx context.Context, payload scaleVMCreateRequest) (*scaleVMResponse, error) {
	var vm scaleVMResponse
	if err := s.doJSON(ctx, http.MethodPost, "/vm/", payload, &vm); err != nil {
		return nil, err
	}
	return &vm, nil
}

func (s *ScaleClientImpl) StartVM(ctx context.Context, vmID string) error {
	return s.doJSON(ctx, http.MethodPost, fmt.Sprintf("/vm/%s/start/", vmID), nil, nil)
}

func (s *ScaleClientImpl) ShutdownVM(ctx context.Context, vmID string) error {
	if err := s.doJSON(ctx, http.MethodPost, fmt.Sprintf("/vm/%s/shutdown/", vmID), nil, nil); err == nil {
		return nil
	} else if IsWorkspaceNotFound(err) {
		return err
	} else {
		return s.doJSON(ctx, http.MethodPost, fmt.Sprintf("/vm/%s/stop/", vmID), nil, nil)
	}
}

func (s *ScaleClientImpl) GetVMStatus(ctx context.Context, vmID string) (string, error) {
	var response scaleVMStatusResponse
	if err := s.doJSON(ctx, http.MethodGet, fmt.Sprintf("/vm/%s/", vmID), nil, &response); err != nil {
		return "", err
	}
	if response.Status != "" {
		return response.Status, nil
	}
	return response.State, nil
}

type scaleVMCreateRequest struct {
	Name        string          `json:"name,omitempty"`
	Description string          `json:"description,omitempty"`
	Bootloader  string          `json:"bootloader,omitempty"`
	Autostart   bool            `json:"autostart,omitempty"`
	VCPUs       int             `json:"vcpus,omitempty"`
	Memory      int             `json:"memory,omitempty"`
	Devices     []scaleVMDevice `json:"devices,omitempty"`
}

type scaleVMDevice struct {
	DType      string         `json:"dtype"`
	Attributes map[string]any `json:"attributes,omitempty"`
}

type scaleVMResponse struct {
	ID     int64  `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	State  string `json:"state,omitempty"`
}

type scaleVMStatusResponse struct {
	Status string `json:"status,omitempty"`
	State  string `json:"state,omitempty"`
}

func newScaleVMCreateRequest(ws contracts.Workspace, cfg scaleRuntimeConfig) scaleVMCreateRequest {
	description := fmt.Sprintf("workspace %s for repo %s", ws.ID, ws.Repo)
	if ws.Owner != "" {
		description = fmt.Sprintf("workspace %s for %s/%s", ws.ID, ws.Owner, ws.Repo)
	}

	devices := []scaleVMDevice{
		{
			DType: "NIC",
			Attributes: map[string]any{
				"type":       "VIRTIO",
				"nic_attach": cfg.Bridge,
			},
		},
		{
			DType: "DISK",
			Attributes: map[string]any{
				"type": "AHCI",
				"path": fmt.Sprintf("/mnt/tank/vm/%s.img", cfg.Name),
				"mode": "AHCI",
				"size": fmt.Sprintf("%dG", cfg.DiskGB),
			},
		},
	}

	if cfg.UserData != "" || len(cfg.SSHAuthorizedKeys) > 0 {
		cloudInit := cfg.UserData
		if cloudInit == "" {
			cloudInit = defaultCloudInit(cfg.SSHAuthorizedKeys)
		}
		devices = append(devices, scaleVMDevice{
			DType: "CLOUDINIT",
			Attributes: map[string]any{
				"user_data": cloudInit,
			},
		})
	}

	return scaleVMCreateRequest{
		Name:        cfg.Name,
		Description: description,
		Bootloader:  "UEFI_CSM",
		Autostart:   true,
		VCPUs:       cfg.VCPUs,
		Memory:      cfg.MemoryMB,
		Devices:     devices,
	}
}

func defaultCloudInit(sshKeys []string) string {
	userData := []string{
		"#cloud-config",
		"package_update: true",
		"packages:",
		"  - docker.io",
		"  - openssh-server",
		"runcmd:",
		"  - [ systemctl, enable, --now, ssh ]",
		"  - [ systemctl, enable, --now, docker ]",
	}
	if len(sshKeys) > 0 {
		userData = append(userData, "ssh_authorized_keys:")
		for _, key := range sshKeys {
			userData = append(userData, fmt.Sprintf("  - %s", key))
		}
	}
	return strings.Join(userData, "\n") + "\n"
}
