FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

# Install Skaffold
RUN curl -Lo /tmp/skaffold "https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64" \
    && install /tmp/skaffold /usr/local/bin/skaffold \
    && rm -f /tmp/skaffold

# Install k3d
RUN curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# Install Telepresence
RUN curl -fL https://app.getambassador.io/download/tel2oss/releases/download/v2.26.2/telepresence-linux-amd64 -o /tmp/telepresence \
    && install /tmp/telepresence /usr/local/bin/telepresence \
    && rm -f /tmp/telepresence

# Pre-configure Docker daemon to trust the k3d registry
RUN mkdir -p /etc/docker \
    && echo '{"insecure-registries": ["k3d-mathtrail-registry.localhost:5050"]}' > /etc/docker/daemon.json
