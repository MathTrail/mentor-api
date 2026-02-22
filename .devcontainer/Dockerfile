# Build swag binary in a Go image (Go is not available in the base image)
FROM golang:1.25-bookworm AS swag-builder
ARG SWAG_VERSION=v2.0.0-rc5
RUN go install github.com/swaggo/swag/v2/cmd/swag@${SWAG_VERSION}

FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04
COPY --from=ghcr.io/mathtrail/platform-env:1 /config/global.env /etc/mathtrail/platform.env

# Install buildah with fuse-overlayfs for nested container builds
RUN apt-get update \
    && apt-get install -y --no-install-recommends buildah fuse-overlayfs \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /etc/containers \
    && printf '[storage]\ndriver = "overlay"\n\n[storage.options.overlay]\nmount_program = "/usr/bin/fuse-overlayfs"\n' \
       > /etc/containers/storage.conf

# Install Skaffold
ARG SKAFFOLD_VERSION=2.17.2
RUN curl -Lo /tmp/skaffold "https://github.com/GoogleContainerTools/skaffold/releases/download/v${SKAFFOLD_VERSION}/skaffold-linux-amd64" \
    && install /tmp/skaffold /usr/local/bin/skaffold \
    && rm -f /tmp/skaffold

# Install Telepresence
ARG TELEPRESENCE_VERSION=2.26.2
RUN curl -fL https://app.getambassador.io/download/tel2oss/releases/download/v${TELEPRESENCE_VERSION}/telepresence-linux-amd64 -o /tmp/telepresence \
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

# Copy swag binary from builder stage
COPY --from=swag-builder /go/bin/swag /usr/local/bin/swag

# Pre-configure buildah to trust the k3d registry and resolve short image names
ARG K3D_REGISTRY=k3d-mathtrail-registry.localhost:5050
RUN mkdir -p /etc/containers/registries.conf.d && \
    printf '[[registry]]\nlocation = "%s"\ninsecure = true\n' "${K3D_REGISTRY}" \
    > /etc/containers/registries.conf.d/k3d.conf && \
    printf 'unqualified-search-registries = ["docker.io"]\n' \
    > /etc/containers/registries.conf
