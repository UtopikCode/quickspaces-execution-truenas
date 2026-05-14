# Repository AI Instructions

This repository implements a Go-based execution adapter for TrueNAS SCALE VM lifecycle management.

Key points:
- The adapter exposes workspace VM lifecycle operations.
- It targets local execution with TrueNAS SCALE API calls and avoids AWS-specific services.
- The codebase is small and centered on `adapter/` package files.
- Tests should use Go's standard tooling: `go test ./...`.

Development guidelines:
- Keep changes idiomatic to Go.
- Preserve VM lifecycle semantics: StartWorkspace, StopWorkspace, GetWorkspaceStatus.
- Use existing project types, helpers, and SCALE API abstractions in `adapter/`.
- Write clear unit tests for adapter logic and edge cases.

Useful commands:
- `go test ./...`
- `go mod tidy`
- `go test ./adapter`

If you need to modify code or tests, prioritize correctness, maintainability, and consistency with the repository's existing structure.
