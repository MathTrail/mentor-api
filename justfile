# MathTrail Mentor Service

set shell := ["bash", "-c"]

NAMESPACE := "mathtrail"
SERVICE := "mathtrail-mentor"

# One-time setup: add Helm repo for service-lib dependency
setup:
    helm repo add mathtrail-charts https://RyazanovAlexander.github.io/mathtrail-charts/charts 2>/dev/null || true
    helm repo update

# Start development mode with hot-reload and port-forwarding
dev: setup
    skaffold dev --port-forward

# Build and deploy to cluster
deploy: setup
    skaffold run

# Remove from cluster
delete:
    skaffold delete

# View pod logs
logs:
    kubectl logs -l app.kubernetes.io/name={{ SERVICE }} -n {{ NAMESPACE }} -f

# Check deployment status
status:
    kubectl get pods -n {{ NAMESPACE }} -l app.kubernetes.io/name={{ SERVICE }}
    kubectl get svc -n {{ NAMESPACE }}

# Connect to cluster and intercept the service for local development
tp-intercept: deploy
    #!/bin/bash
    set -e
    telepresence connect
    telepresence intercept {{ SERVICE }} --namespace {{ NAMESPACE }} --port 8080:8080
    echo "Intercepting {{ SERVICE }}. Run your service locally: go run ."

# Stop intercept and disconnect
tp-stop:
    telepresence leave {{ SERVICE }}-{{ NAMESPACE }} 2>/dev/null || true
    telepresence quit

# Show Telepresence status
tp-status:
    telepresence status
    telepresence list --namespace {{ NAMESPACE }}

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
