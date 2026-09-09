# ---------------------------------------------------------------------------
# Stage 1: Build code-reviewer and Localharness
# ---------------------------------------------------------------------------
FROM --platform=$BUILDPLATFORM golang:1.25-bookworm AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

RUN git clone --depth 1 --branch v0.3.0 https://github.com/divmora/localharness.git /tmp/localharness \
    && cd /tmp/localharness \
    && CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -o /go/bin/localharness ./cmd/localharness \
    && rm -rf /tmp/localharness

COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /code-reviewer .

# ---------------------------------------------------------------------------
# Stage 2: Runtime Image (Minimal, Secure Debian Slim)
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim

ARG BUILD_DATE
ARG VCS_REF
ARG VERSION="dev"

LABEL org.opencontainers.image.title="code-reviewer-ai-agent" \
    org.opencontainers.image.description="Autonomous AI Code Reviewer Agent powered by Divmora LocalHarness SDK" \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.created="${BUILD_DATE}" \
    org.opencontainers.image.revision="${VCS_REF}" \
    org.opencontainers.image.source="https://github.com/divmora/code-reviewer-ai-agent" \
    org.opencontainers.image.vendor="Divmora"

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    git \
    curl \
    jq \
    ripgrep \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /code-reviewer /usr/local/bin/code-reviewer
COPY --from=builder /go/bin/localharness /usr/local/bin/localharness

WORKDIR /workspace

# Trust all directories for git operations in containers
RUN git config --system --add safe.directory '*'

ENTRYPOINT ["/usr/local/bin/code-reviewer"]
CMD ["--help"]
