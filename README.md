# quickspaces-execution-truenas

A local execution adapter for TrueNAS and Docker/Podman environments, implemented in Go.

## Overview

This repository contains a Go implementation of an `ExecutionAdapter` that manages workspace containers with Docker-compatible APIs.

Supported operations:

- `StartWorkspace` → container creation and start (`docker run` semantics)
- `StopWorkspace` → container stop (`docker stop` semantics)
- `GetWorkspaceStatus` → container inspection (`docker inspect` semantics)

The adapter is designed for local TrueNAS execution and avoids any AWS dependency.

## TrueNAS Setup

1. Use TrueNAS SCALE with Docker or a Docker-compatible runtime.
2. Ensure the Docker or Podman socket is available to the adapter environment.
   - Docker: `/var/run/docker.sock`
   - Podman: `unix:///run/podman/podman.sock`
3. If using Podman, set:

```bash
export DOCKER_HOST=unix:///run/podman/podman.sock
```

4. Mount the socket into the container or runtime where this adapter is executed.

## Development

Build and test the project locally:

```bash
go test ./...
```

## Devcontainer

The repository includes a `.devcontainer` configuration with the Docker CLI installed.
The devcontainer mounts the Docker socket so you can run containers from inside VS Code.

## CI

A GitHub Actions workflow is included to run tests on push and pull request events.
