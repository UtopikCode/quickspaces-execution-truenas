# AI Instructions

This repository implements a local execution adapter for TrueNAS SCALE VM management in Go.

Requirements:
- Implement `ExecutionAdapter`
- Use TrueNAS SCALE API calls for VM lifecycle operations
- Do not depend on AWS
- `StartWorkspace` must perform VM creation and start via SCALE
- `StopWorkspace` must stop/shutdown the VM via SCALE
- `GetWorkspaceStatus` must inspect VM state via SCALE
- Include tests with a mock SCALE client
- Include GitHub Actions CI
