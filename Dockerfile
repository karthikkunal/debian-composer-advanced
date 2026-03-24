# Multi-stage build for minimal image
FROM golang:1.21-alpine AS builder

WORKDIR /build

# Install dependencies
RUN apk add --no-cache git make

# Copy go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build
RUN make build

# Runtime image
FROM debian:bookworm-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    yq \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Install snapper for snapshots
RUN apt-get update && apt-get install -y --no-install-recommends snapper btrfs-progs || true \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /build/bin/debian-composer-go /usr/local/bin/debian-composer

# Copy kitchen
COPY --from=builder /build/kitchen /usr/share/debian-composer/kitchen

# Create state directory
RUN mkdir -p /var/lib/debian-composer

# Symlink kitchen to default location
RUN ln -sf /usr/share/debian-composer/kitchen /workspace/kitchen

WORKDIR /workspace

ENTRYPOINT ["debian-composer"]
CMD ["--help"]
