# Repository AI Instructions

This repository implements a Go-based execution adapter for TrueNAS and Docker/Podman environments.

Key points:
- The adapter exposes workspace container lifecycle operations.
- It targets local execution with Docker-compatible APIs and avoids AWS-specific services.
- The codebase is small and centered on `adapter/` package files.
- Tests should use Go's standard tooling: `go test ./...`.

Development guidelines:
- Keep changes idiomatic to Go.
- Preserve container management semantics: StartWorkspace, StopWorkspace, GetWorkspaceStatus.
- Use existing project types, helpers, and Docker client abstractions in `adapter/`.
- Write clear unit tests for adapter logic and edge cases.

Useful commands:
- `go test ./...`
- `go mod tidy`
- `go test ./adapter`

If you need to modify code or tests, prioritize correctness, maintainability, and consistency with the repository's existing structure.
