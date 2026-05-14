# quickspaces-execution-truenas

A local execution adapter for TrueNAS SCALE, implemented in Go.

## Overview

This repository contains a Go implementation of an `ExecutionAdapter` that manages workspace VMs through the TrueNAS SCALE API.

Supported operations:

- `StartWorkspace` → VM creation and start via SCALE
- `StopWorkspace` → VM shutdown via SCALE
- `GetWorkspaceStatus` → VM state inspection via SCALE

The adapter is designed for local TrueNAS SCALE execution and avoids any AWS dependency.

## TrueNAS SCALE Setup

The adapter uses the TrueNAS SCALE REST API to create and manage VMs.

### SCALE host configuration

Example `hostConfig` for SCALE API access:

```json
{
  "host": "https://truenas.local",
  "apiToken": "YOUR_API_TOKEN",
  "insecure": true
}
```

### VM runtime config and profiles

A runtime config can define an image, resources, networking, and SSH keys. Named VM profiles are also supported.

Example using a named profile:

```json
{
  "image": "ubuntu-22.04-cloudimg-amd64.qcow2",
  "profile": "small",
  "sshAuthorizedKeys": [
    "ssh-rsa AAAAB3Nza... user@example.com"
  ]
}
```

Profiles provide default resource values when specific settings are omitted:

- `small` → 2 vCPUs, 4 GiB memory, 20 GiB disk
- `medium` → 4 vCPUs, 8 GiB memory, 40 GiB disk
- `large` → 8 vCPUs, 16 GiB memory, 80 GiB disk

Example overriding profile defaults:

```json
{
  "image": "ubuntu-22.04-cloudimg-amd64.qcow2",
  "profile": "small",
  "memoryMB": 8192,
  "diskGB": 40,
  "sshAuthorizedKeys": [
    "ssh-rsa AAAAB3Nza... user@example.com"
  ]
}
```

The adapter also supports direct resource configuration without a profile.

### Cloud-init provisioning

The SCALE adapter can inject a cloud-init payload that installs Docker and enables SSH if the guest image supports cloud-init.

## Development

Build and test the project locally:

```bash
go test ./...
```

## CI

A GitHub Actions workflow is included to run tests on push and pull request events.

Build and test the project locally:

```bash
go test ./...
```

## Devcontainer

The repository includes a `.devcontainer` configuration for local Go development.

## CI

A GitHub Actions workflow is included to run tests on push and pull request events.
