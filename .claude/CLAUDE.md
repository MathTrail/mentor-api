# MathTrail Mentor Service

## Overview

Student Feedback Loop service for the MathTrail platform. Receives student feedback, analyzes it (sentiment, difficulty), persists it to PostgreSQL via Dapr binding, and exposes an AI mentor strategy update in the response.

**Language:** Go 1.25.7
**Port:** 8080
**Cluster:** k3d `mathtrail-dev`, namespace `mathtrail`
**KUBECONFIG:** `/home/vscode/.kube/k3d-mathtrail-dev.yaml`

## Tech Stack

| Layer | Library |
|-------|---------|
| HTTP | `github.com/gin-gonic/gin` |
| Database | `github.com/jackc/pgx/v5` via Dapr binding (`internal/database/dapr_binding.go`) |
| Config | `github.com/spf13/viper` |
| Logging | `go.uber.org/zap` |
| Tracing | OpenTelemetry OTLP gRPC → Tempo |
| Metrics | OpenTelemetry → Prometheus exporter (`/metrics`) |
| Profiling | Pyroscope (`github.com/grafana/pyroscope-go`) |
| Dapr | `github.com/dapr/go-sdk` — DB binding + sidecar config |
| Swagger | `github.com/swaggo/gin-swagger` — served at `/swagger/*any` |

## Key Files

| File | Purpose |
|------|---------|
| `cmd/server/main.go` | HTTP server entry point |
| `cmd/migrate/main.go` | DB migration runner |
| `migrations/001_init.sql` | Initial schema |
| `internal/app/container.go` | Dependency injection container |
| `internal/config/config.go` | Config (Viper) |
| `internal/database/dapr_binding.go` | PostgreSQL via Dapr output binding |
| `internal/feedback/` | model, repository, service, controller |
| `internal/server/router.go` | Gin router with all routes & middleware |
| `internal/server/middleware.go` | RequestID, ZapLogger, ZapRecovery, UserSpanAttributes |
| `internal/observability/observability.go` | InitTracer, InitMetrics, InitPyroscope |
| `internal/clients/llm_client.go` | LLM client (future v2) |
| `internal/clients/profile_client.go` | Profile service client |
| `infra/helm/mentor-api/` | Helm chart (uses `mathtrail-service-lib`) |
| `infra/helm/values-dev.yaml` | Dev environment values |
| `infra/helm/values-observability.yaml` | Observability profile values |
| `skaffold.yaml` | Skaffold pipeline config |
| `justfile` | Build, deploy, test automation |
| `.devcontainer/devcontainer.json` | VS Code devcontainer config |

## API

```
POST   /api/v1/feedback     — submit student feedback → returns StrategyUpdate
GET    /health/startup       — Kubernetes startup probe
GET    /health/liveness      — Kubernetes liveness probe
GET    /health/ready         — readiness probe (checks DB via Dapr binding)
GET    /metrics              — Prometheus metrics
GET    /dapr/config          — Dapr sidecar config (returns {}, suppresses 404s)
GET    /swagger/*any         — Swagger UI
```

## Architecture

- **DB access:** Dapr output binding (`postgres` component) — no direct pgx connection in app code
- **CDC:** Debezium monitors the `feedback` table, publishes events to Kafka — app does NOT publish events
- **Dapr App ID:** `mentor-api`
- Helm chart uses `mathtrail-service-lib` library chart from `https://MathTrail.github.io/charts/charts`
- The service-lib provides: ServiceAccount, RBAC, ConfigMap, Migration Job, Deployment, Service, HPA

## Service-Lib Contract (MUST follow)

- **Health probes required:** `/health/startup`, `/health/liveness`, `/health/ready`
- **Security:** Container must run as non-root (UID 10001), `readOnlyRootFilesystem: true`
- **Validation:** `image.repository`, `image.tag`, `resources.requests`, `resources.limits` must be defined in values.yaml

## Development Workflow

```bash
just dev                       # Skaffold dev mode: hot-reload + port-forward
just dev observability=true    # dev mode with observability profile
just deploy                    # One-time build and deploy
just delete                    # Remove from cluster
just test                      # go test ./... -v
just swagger                   # Regenerate Swagger docs (swag init)
just logs                      # View pod logs
just status                    # Check pods and services
just load-test                 # Run k6 load tests
```

## Development Standards

- Follow Clean Architecture: Domain → Repository → Service → Controller
- Handle errors explicitly — never ignore error returns
- All comments in English
- Middleware order: otelgin → UserSpanAttributes → RequestID → ZapLogger → ZapRecovery
- Commit convention: `feat(feedback):`, `fix(feedback):`, `test(feedback):`, `docs(feedback):`

## External Dependencies

| Repo | Purpose |
|------|---------|
| `mathtrail-charts` | Hosts `mathtrail-service-lib` library chart |
| `mathtrail-infra-local-k3s` | k3d cluster setup |
| `mathtrail-profile` | Reference implementation using same service-lib pattern |

## Pre-requisites (run on host before opening devcontainer)

```bash
# In mathtrail-infra-local-k3s repo:
just create        # Creates k3d cluster
just kubeconfig    # Saves kubeconfig to ~/.kube/k3d-mathtrail-dev.yaml
```
