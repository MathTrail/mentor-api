# Implementation Plan: Deploy mathtrail-mentor to k3s

Execute these steps in order to make the helm chart deployable.

---

## Step 1: Add health endpoints to `cmd/main.go`

The service-lib probes expect `/health/startup`, `/health/liveness`, `/health/ready`. Currently only `/hello` exists.

Add three handlers using stdlib `net/http` (no new dependencies):

```go
package main

import (
	"encoding/json"
	"log"
	"net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Hello, MathTrail!"})
}

func healthStartup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

func healthLiveness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func healthReady(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

func main() {
	http.HandleFunc("/hello", helloHandler)
	http.HandleFunc("/health/startup", healthStartup)
	http.HandleFunc("/health/liveness", healthLiveness)
	http.HandleFunc("/health/ready", healthReady)

	log.Println("Starting mathtrail-mentor on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
```

---

## Step 2: Fix Dockerfile for non-root execution

The service-lib enforces `runAsNonRoot: true` and `readOnlyRootFilesystem: true`. Current Dockerfile runs as root.

Replace entire `Dockerfile` with:

```dockerfile
FROM golang:1.25.7-alpine AS builder

RUN apk add --no-cache git

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ ./cmd/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/main.go

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -g 10001 -S appgroup \
    && adduser -u 10001 -S appuser -G appgroup

COPY --from=builder /server /server

USER 10001

EXPOSE 8080
ENTRYPOINT ["/server"]
```

Also update `go.mod` to use Go 1.25.7:
```
module mathtrail-mentor

go 1.25.7
```

---

## Step 3: Fix `.devcontainer/devcontainer.json`

Replace with:

```json
{
    "name": "MathTrail Mentor DevContainer",
    "workspaceFolder": "/workspace/mathtrail-mentor",
    "features": {
        "ghcr.io/devcontainers/features/go:1": {
            "version": "1.25.7"
        },
        "ghcr.io/devcontainers/features/docker-in-docker:2": {},
        "ghcr.io/devcontainers/features/kubectl-helm-minikube:1.2.2": {
            "helm": "3.14.0",
            "version": "1.31.0",
            "minikube": "none"
        },
        "ghcr.io/eitsupi/devcontainer-features/just:0.1.1": {},
        "ghcr.io/dapr/cli/dapr-cli:0": {}
    },
    "mounts": [
        "source=${localEnv:HOME}${localEnv:USERPROFILE}/.kube,target=/home/vscode/.kube,type=bind,consistency=cached"
    ],
    "remoteEnv": {
        "KUBECONFIG": "/home/vscode/.kube/k3d-mathtrail-dev.yaml"
    },
    "customizations": {
        "vscode": {
            "extensions": [
                "golang.go",
                "ms-azuretools.vscode-docker",
                "ms-kubernetes-tools.vscode-kubernetes-tools",
                "ms-kubernetes.helm",
                "redhat.vscode-yaml",
                "eamodio.gitlens",
                "sclu1034.justfile",
                "Tim-Koehler.helm-intellisense",
                "anthropic.claude-code"
            ]
        }
    },
    "postCreateCommand": "bash -c 'set -e && mkdir -p /home/vscode/.kube && chmod 700 /home/vscode/.kube && go mod download && helm repo add mathtrail-charts https://RyazanovAlexander.github.io/mathtrail-charts/charts && helm dependency build helm/mathtrail-mentor && kubectl cluster-info 2>/dev/null && echo \"Ready to deploy to K3d cluster\" || echo \"Cluster not accessible. Run just create on host first\"'",
    "forwardPorts": [8080]
}
```

Fixes: Go feature syntax, Helm 3.14.0 (4.1.0 doesn't exist), kubectl 1.31.0, cross-platform kube mount, postCreateCommand with dependency setup, forwardPorts, additional extensions.

---

## Step 4: Rewrite `justfile`

Replace with:

```just
# MathTrail Mentor Service

set shell := ["bash", "-c"]

NAMESPACE := "mathtrail"
SERVICE := "mathtrail-mentor"
CLUSTER := "mathtrail-dev"
IMAGE := "mathtrail-mentor:latest"

# Build Docker image
build:
    docker build -t {{ IMAGE }} .

# Import image into k3d cluster
import: build
    k3d image import {{ IMAGE }} -c {{ CLUSTER }}

# Build Helm chart dependencies
deps:
    helm repo add mathtrail-charts https://RyazanovAlexander.github.io/mathtrail-charts/charts 2>/dev/null || true
    helm dependency build helm/{{ SERVICE }}

# Deploy to k3d cluster (builds, imports, and installs)
deploy: import deps
    helm upgrade --install {{ SERVICE }} ./helm/{{ SERVICE }} \
        --namespace {{ NAMESPACE }} \
        --create-namespace \
        --values ./helm/{{ SERVICE }}/values.yaml \
        --wait --timeout 120s
    kubectl get pods -n {{ NAMESPACE }} -l app.kubernetes.io/name={{ SERVICE }}

# Remove from cluster
delete:
    helm uninstall {{ SERVICE }} -n {{ NAMESPACE }} || true

# View pod logs
logs:
    kubectl logs -l app.kubernetes.io/name={{ SERVICE }} -n {{ NAMESPACE }} -f

# Check deployment status
status:
    kubectl get pods -n {{ NAMESPACE }} -l app.kubernetes.io/name={{ SERVICE }}
    kubectl get svc -n {{ NAMESPACE }}

# Test endpoints via port-forward
test:
    #!/bin/bash
    set -e
    echo "Starting port-forward..."
    kubectl port-forward svc/{{ SERVICE }} 8080:8080 -n {{ NAMESPACE }} &
    PF_PID=$!
    trap "kill $PF_PID 2>/dev/null" EXIT
    sleep 3
    echo "Testing /hello..."
    curl -s http://localhost:8080/hello | jq .
    echo ""
    echo "Testing /health/ready..."
    curl -s http://localhost:8080/health/ready | jq .
```

Critical additions: `k3d image import` (without this k3s can't pull local images), `helm dependency build`, namespace `mathtrail`, correct port `8080:8080`.

---

## Step 5: Verify deployment

```bash
kubectl cluster-info                    # Verify cluster access
just deploy                             # Full pipeline: build -> import -> deps -> helm install
kubectl get pods -n mathtrail           # Should show Running, 1/1 READY
just test                               # curl /hello and /health/ready
just logs                               # Check for startup message
```

If pod is in `CrashLoopBackOff`:
```bash
kubectl describe pod -n mathtrail -l app.kubernetes.io/name=mathtrail-mentor
```

| Symptom | Cause | Fix |
|---------|-------|-----|
| `CreateContainerConfigError` | Image runs as root | Check `USER 10001` in Dockerfile |
| `CrashLoopBackOff` + probe failures | Health endpoints missing | Verify `/health/startup` works in Docker |
| `ImagePullBackOff` | Image not in k3d | Run `just import` |
| Helm dependency errors | Missing `helm dependency build` | Run `just deps` |
