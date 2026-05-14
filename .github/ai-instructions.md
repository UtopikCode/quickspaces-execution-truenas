# AI Instructions

This repository implements a local execution adapter for TrueNAS / Docker environments in Go.

Requirements:
- Implement `ExecutionAdapter`
- Use Docker or Podman APIs
- Do not depend on AWS
- `StartWorkspace` must perform a container creation and start (`docker run` semantics)
- `StopWorkspace` must stop the container (`docker stop` semantics)
- `GetWorkspaceStatus` must inspect the container (`docker inspect` semantics)
- Include tests with a mock Docker client
- Include a devcontainer setup with Docker CLI
- Include GitHub Actions CI
