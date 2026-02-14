# MathTrail Mentor Service

## Overview

Intelligent learning service for MathTrail platform — an AI mentor that delivers math olympiad challenges, analyzes progress, and powers the student learning flow.

**Language:** Go 1.25.7 (stdlib only, no external dependencies)
**Port:** 8080
**Cluster:** k3d `mathtrail-dev`, namespace `mathtrail`
**KUBECONFIG:** `/home/vscode/.kube/k3d-mathtrail-dev.yaml`

## Key Files

| File | Purpose |
|------|---------|
| `cmd/main.go` | HTTP server entry point |
| `Dockerfile` | Multi-stage Docker build |
| `helm/mathtrail-mentor/Chart.yaml` | Helm chart metadata, depends on `mathtrail-service-lib` |
| `helm/mathtrail-mentor/values.yaml` | Deployment configuration |
| `helm/mathtrail-mentor/templates/all.yaml` | Includes all service-lib templates |
| `skaffold.yaml` | Skaffold pipeline config (build + Helm deploy) |
| `justfile` | Build, deploy, test automation (wraps Skaffold) |
| `.devcontainer/devcontainer.json` | VS Code devcontainer config |

## Architecture

- Helm chart uses `mathtrail-service-lib` library chart from `https://MathTrail.github.io/charts/charts`
- The service-lib provides: ServiceAccount, RBAC, ConfigMap, Migration Job, Deployment, Service, HPA
- Dapr sidecar integration enabled

## Service-Lib Contract (MUST follow)

- **Health probes required:** `/health/startup`, `/health/liveness`, `/health/ready`
- **Security:** Container must run as non-root (UID 10001), `readOnlyRootFilesystem: true`
- **Validation:** `image.repository`, `image.tag`, `resources.requests`, `resources.limits` must be defined in values.yaml

## Development Workflow

Uses **Skaffold + Helm** for build and deploy. Skaffold handles Docker build, k3d image loading, Helm dependency resolution, and Helm install/upgrade automatically.

```bash
just dev      # Skaffold dev mode: hot-reload + port-forward (Ctrl+C to stop)
just deploy   # One-time build and deploy (skaffold run)
just delete   # Remove from cluster (skaffold delete)
just test     # Port-forward and test endpoints
just logs     # View pod logs
just status   # Check pods and services
```

## External Dependencies

| Repo | Purpose |
|------|---------|
| `mathtrail-charts` | Hosts `mathtrail-service-lib` library chart (fetched via Helm repo, no local access needed) |
| `mathtrail-infra-local-k3s` | k3d cluster setup — run `just create` + `just kubeconfig` on host before using devcontainer |
| `mathtrail-profile` | Reference implementation using same service-lib pattern |

## Pre-requisites (run on host before opening devcontainer)

```bash
# In mathtrail-infra-local-k3s repo:
just create        # Creates k3d cluster
just kubeconfig    # Saves kubeconfig to ~/.kube/k3d-mathtrail-dev.yaml
```
