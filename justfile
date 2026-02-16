# MathTrail Mentor Service

set shell := ["bash", "-c"]

NAMESPACE := "mathtrail"
SERVICE := "mentor-api"
CHART_NAME := "mentor-api"
TEST_NAMESPACE := "mathtrail"
TEST_CONFIGMAP := "mentor-api-functional"

# -- Portable Image Build (buildctl → buildah) --------------------------------

# Build and push a container image.
# Called by Skaffold via: just build-push-image
# Uses $IMAGE env var set by Skaffold, or accepts a tag argument.
# CI (buildctl available): uses BuildKit
# Local dev:               uses buildah
build-push-image tag=env("IMAGE", ""):
    #!/bin/bash
    set -euo pipefail
    TAG="{{tag}}"
    if [ -z "$TAG" ]; then
        echo "Error: no image tag provided (set \$IMAGE or pass as argument)" >&2
        exit 1
    fi
    if [ "${CI:-}" = "true" ] || command -v buildctl &>/dev/null; then
        echo "🔨 Building with BuildKit..."
        buildctl build \
            --frontend=dockerfile.v0 \
            --local context=. \
            --local dockerfile=. \
            --output type=image,name="$TAG",push=true,registry.insecure=true \
            --export-cache type=inline \
            --import-cache type=registry,ref="$TAG"
    else
        echo "🔨 Building with Buildah..."
        buildah bud --tag "$TAG" .
        buildah push --tls-verify=false "$TAG"
    fi

# -- Development ---------------------------------------------------------------

# One-time setup: add Helm repo for service-lib dependency
setup:
    helm repo add mathtrail-charts https://MathTrail.github.io/charts/charts 2>/dev/null || true
    helm repo update

# Build the Go binary
build:
    go build -o bin/server ./cmd/server

# Run all tests
test:
    go test ./... -v

# Run load tests: bundle scripts with esbuild, deploy k6-test-runner from OCI
load-test:
    #!/bin/bash
    set -euo pipefail
    mkdir -p tests/load/dist
    esbuild tests/load/scripts/main.js \
        --bundle \
        --format=esm \
        --external:k6 \
        --external:'k6/*' \
        --outfile=tests/load/dist/bundle.js
    skaffold run -p load-test --tail

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

# -- CI/CD Contract (called by self-hosted runner) ----------------------------

# Lint the codebase
ci-lint:
    golangci-lint run ./...

# Run tests
ci-test ns="":
    #!/bin/bash
    set -e
    if [ -n "{{ns}}" ]; then
        NAMESPACE={{ns}} go test ./... -v -count=1
    else
        go test ./... -v -count=1
    fi

# Fast binary build for PR verification
ci-build:
    go build -o bin/server ./cmd/main.go

# Create an ephemeral namespace for a PR
ci-prepare ns:
    #!/bin/bash
    set -e
    kubectl create namespace {{ns}} 2>/dev/null || true
    kubectl label namespace {{ns}} app.kubernetes.io/managed-by=ci --overwrite

# Delete an ephemeral namespace
ci-cleanup ns:
    kubectl delete namespace {{ns}} --wait=false 2>/dev/null || true

# -- Chart Release (OCI Version) ----------------------------------------------

# Package and publish chart to OCI registry
# Usage: just release-chart-oci oci://my-registry.com/charts
release-chart-oci registry_url="oci://k3d-mathtrail-registry.localhost:5050/charts":
    #!/bin/bash
    set -e
    CHART_DIR="infra/helm/{{ CHART_NAME }}"
    
    # Automatically extract version from Chart.yaml
    VERSION=$(grep '^version:' "$CHART_DIR/Chart.yaml" | awk '{print $2}')
    
    echo "📦 Packaging {{ CHART_NAME }} v${VERSION}..."
    helm package "$CHART_DIR" --destination /tmp/charts
    
    echo "🚀 Pushing to OCI registry: {{ registry_url }}..."
    # Helm OCI uses the repository path rather than the specific file name in the URL
    helm push "/tmp/charts/{{ CHART_NAME }}-${VERSION}.tgz" "{{ registry_url }}"
    
    echo "✅ Published {{ CHART_NAME }} v${VERSION} to OCI"

# -- Telepresence --------------------------------------------------------------

# Connect to cluster and intercept the service for local development
tp-intercept: deploy
    #!/bin/bash
    set -e
    telepresence connect -n {{ NAMESPACE }}
    telepresence intercept {{ SERVICE }} --port 8080:8080
    echo "Intercepting {{ SERVICE }}. Run your service locally: go run ./cmd/main.go"

# Stop intercept and disconnect
tp-stop:
    telepresence leave {{ SERVICE }} 2>/dev/null || true
    telepresence quit

# Show Telepresence status
tp-status:
    telepresence status
    telepresence list

# Test endpoints via port-forward
test-endpoints:
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

# -- Chart Release -------------------------------------------------------------

# Package and publish chart to mathtrail-charts
release-chart:
    #!/bin/bash
    set -e
    CHART_DIR="infra/helm/{{ CHART_NAME }}"
    VERSION=$(grep '^version:' "$CHART_DIR/Chart.yaml" | awk '{print $2}')
    echo "Packaging {{ CHART_NAME }} v${VERSION}..."
    helm package "$CHART_DIR" --destination /tmp/mathtrail-charts

    CHARTS_REPO="/tmp/mathtrail-charts-repo"
    rm -rf "$CHARTS_REPO"
    git clone git@github.com:MathTrail/charts.git "$CHARTS_REPO"
    cp /tmp/mathtrail-charts/{{ CHART_NAME }}-*.tgz "$CHARTS_REPO/charts/"
    cd "$CHARTS_REPO"
    helm repo index ./charts \
        --url https://MathTrail.github.io/charts/charts
    git add charts/
    git commit -m "chore: release {{ CHART_NAME }} v${VERSION}"
    git push
    echo "Published {{ CHART_NAME }} v${VERSION} to mathtrail-charts"

# -- Terraform -----------------------------------------------------------------

# Initialize Terraform for an environment
tf-init ENV:
    cd infra/terraform/environments/{{ ENV }} && terraform init

# Plan Terraform changes
tf-plan ENV:
    cd infra/terraform/environments/{{ ENV }} && terraform plan

# Apply Terraform changes
tf-apply ENV:
    cd infra/terraform/environments/{{ ENV }} && terraform apply

# -- On-prem Node Preparation -------------------------------------------------

# Prepare an Ubuntu node for on-prem deployment
prepare-node IP:
    cd infra/ansible && ansible-playbook \
        -i "{{ IP }}," \
        playbooks/setup.yml
