FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

# Install buildah with fuse-overlayfs for nested container builds
RUN apt-get update \
    && apt-get install -y --no-install-recommends buildah fuse-overlayfs \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /etc/containers \
    && printf '[storage]\ndriver = "overlay"\n\n[storage.options.overlay]\nmount_program = "/usr/bin/fuse-overlayfs"\n' \
       > /etc/containers/storage.conf

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

# Install BuildKit CLI (buildctl — talks to buildkitd sidecar in K3s)
ARG BUILDKIT_VERSION=0.27.1
RUN curl -sSfL "https://github.com/moby/buildkit/releases/download/v${BUILDKIT_VERSION}/buildkit-v${BUILDKIT_VERSION}.linux-amd64.tar.gz" \
    | tar xz -C /usr/local/bin --strip-components=1 bin/buildctl

# Install esbuild (for bundling k6 load test scripts)
ARG ESBUILD_VERSION=0.27.3
RUN curl -fsSL "https://registry.npmjs.org/@esbuild/linux-x64/-/linux-x64-${ESBUILD_VERSION}.tgz" \
    | tar xz --strip-components=2 -C /usr/local/bin package/bin/esbuild

# Pre-configure buildah to trust the k3d registry and resolve short image names
RUN mkdir -p /etc/containers/registries.conf.d && \
    printf '[[registry]]\nlocation = "k3d-mathtrail-registry.localhost:5050"\ninsecure = true\n' \
    > /etc/containers/registries.conf.d/k3d.conf && \
    printf 'unqualified-search-registries = ["docker.io"]\n' \
    > /etc/containers/registries.conf
