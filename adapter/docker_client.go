package adapter

import (
	"context"
	"io"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
)

type DockerClient interface {
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error
	ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error
	ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error)
	ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error)
}

type DockerClientImpl struct {
	client *client.Client
}

func NewDockerClient() (DockerClient, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerClientImpl{client: c}, nil
}

func (d *DockerClientImpl) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, containerName string) (container.CreateResponse, error) {
	return d.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, containerName)
}

func (d *DockerClientImpl) ContainerStart(ctx context.Context, containerID string, options types.ContainerStartOptions) error {
	return d.client.ContainerStart(ctx, containerID, options)
}

func (d *DockerClientImpl) ContainerStop(ctx context.Context, containerID string, timeout *time.Duration) error {
	return d.client.ContainerStop(ctx, containerID, timeout)
}

func (d *DockerClientImpl) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return d.client.ContainerInspect(ctx, containerID)
}

func (d *DockerClientImpl) ImagePull(ctx context.Context, ref string, options types.ImagePullOptions) (io.ReadCloser, error) {
	return d.client.ImagePull(ctx, ref, options)
}
