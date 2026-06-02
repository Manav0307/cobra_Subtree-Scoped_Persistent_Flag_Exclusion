# Benchmark-compliant offline development and test environment for the Cobra challenge.
# Debian-based, root-default, with pre-warmed Go module/build caches.

ARG GO_VERSION=1.22.10
ARG BASE_COMMIT=ad460ea8f249db69c943a365fb84f3a59042d54e

FROM debian:bookworm-slim

ARG GO_VERSION
ARG BASE_COMMIT

ENV DEBIAN_FRONTEND=noninteractive \
    GO111MODULE=on \
    CGO_ENABLED=0 \
    GOPATH=/go \
    GOMODCACHE=/gomod/cache \
    GOCACHE=/gocache/build \
    PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

# --- Build-time network access: install toolchain and warm caches only here ---
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash \
        ca-certificates \
        curl \
        git \
        patch \
        gawk \
        build-essential \
    && rm -rf /var/lib/apt/lists/*

# Official Go toolchain (matches CI minimum used by golangci-lint workflow).
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" \
    | tar -C /usr/local -xz

# Non-root user required by benchmark test.sh root-isolation patterns.
RUN useradd -m -s /bin/bash _gotest

# Shared module cache and writable build caches for root and _gotest.
RUN mkdir -p "${GOMODCACHE}" "${GOCACHE}" /tmp/gocache_gotest \
    && chown -R _gotest:_gotest /tmp/gocache_gotest

WORKDIR /workspace

# Copy repository (including .git for patch application and history inspection).
COPY . .

# Reset to the challenge base commit so runtime patches apply cleanly.
RUN git checkout --force "${BASE_COMMIT}" \
    && git clean -fdx

# Pre-populate module and build caches while network is available.
RUN go mod download \
    && go test -count=0 ./... \
    && chmod -R a+rX "${GOMODCACHE}" "${GOCACHE}"

# After this point the container must not reach the network for Go/module fetches.
ENV GOPROXY=off \
    GOSUMDB=off

# Sanity check: verify offline test execution at base commit.
RUN go test -count=1 ./... > /dev/null

WORKDIR /workspace

# Default to root to mirror platform benchmark execution.
USER root

CMD ["/bin/bash"]
