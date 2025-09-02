# Build stage
FROM --platform=$BUILDPLATFORM golang:1.22-bookworm AS builder

# Install security updates and dependencies
RUN apt-get update && apt-get install -y \
    ca-certificates \
    git \
    gcc \
    libc6-dev \
    libsqlite3-dev \
    gcc-x86-64-linux-gnu \
    libc6-dev-amd64-cross \
    && rm -rf /var/lib/apt/lists/*

# Arguments for cross-compilation
ARG TARGETOS
ARG TARGETARCH

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=$TARGETOS GOARCH=$TARGETARCH CC=x86_64-linux-gnu-gcc go build \
    -ldflags='-w -s' \
    -o carryless main.go

# Create data directory for database
RUN mkdir -p /tmp/data && chown 65532:65532 /tmp/data

# Final stage
FROM gcr.io/distroless/base-debian12:nonroot

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder --chown=nonroot:nonroot /app/carryless .

# Copy static assets and templates
COPY --from=builder --chown=nonroot:nonroot /app/static ./static
COPY --from=builder --chown=nonroot:nonroot /app/templates ./templates

# Copy and create data directory
COPY --from=builder --chown=nonroot:nonroot /tmp/data /data

# Expose port
EXPOSE 8080

# Run the application
CMD ["./carryless"]
