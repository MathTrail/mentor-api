# MathTrail Mentor Service

set shell := ["bash", "-c"]

NAMESPACE := "mathtrail"
SERVICE := "mathtrail-mentor"
CHART_NAME := "mathtrail-mentor"

# -- Development ---------------------------------------------------------------

# One-time setup: add Helm repo for service-lib dependency
setup:
    helm repo add mathtrail-charts https://MathTrail.github.io/mathtrail-charts/charts 2>/dev/null || true
    helm repo update

# Build the Go binary
build:
    go build -o bin/server ./cmd/server

# Run all tests
test:
    go test ./... -v

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
    git clone git@github.com:MathTrail/mathtrail-charts.git "$CHARTS_REPO"
    cp /tmp/mathtrail-charts/{{ CHART_NAME }}-*.tgz "$CHARTS_REPO/charts/"
    cd "$CHARTS_REPO"
    helm repo index ./charts \
        --url https://MathTrail.github.io/mathtrail-charts/charts
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
