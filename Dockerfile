# -------- Stage 1: Build the binary --------
# Use Go Alpine image (lightweight) for building
FROM golang:1.25-alpine AS builder

# Set working directory for build
WORKDIR /build

# Copy Go dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build a statically linked binary for Linux/amd64
# - CGO disabled to avoid C dependencies
# - ldflags reduce binary size
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -o trafficctrl \
    ./cmd/ctrl

# -------- Stage 2: Final runtime image --------
FROM alpine:latest

# Install certs & timezone data (needed for HTTPS + logging timestamps)
RUN apk --no-cache add ca-certificates tzdata

# Set working directory inside the container
WORKDIR /app

# Copy the built binary from stage 1
COPY --from=builder /build/trafficctrl /app/trafficctrl

# Copy default YAML configs (can be overridden by mounting)
COPY --from=builder /build/config/*.yaml /app/config/

# Defines where the Config loader looks
ENV CONFIG_DIR=/app/config

# Create non-root user for security
RUN addgroup -g 1000 trafficctrl && \
    adduser -D -u 1000 -G trafficctrl trafficctrl && \
    chown -R trafficctrl:trafficctrl /app

# Switch to non-root user
USER trafficctrl

# Expose proxy + metrics ports
EXPOSE 8080 8090

# Run the proxy binary
ENTRYPOINT ["/app/trafficctrl"]
