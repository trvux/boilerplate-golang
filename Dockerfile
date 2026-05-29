# Stage 1: Build the static Go binary
FROM golang:1.26-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Cache dependency layer
COPY go.mod go.sum ./
RUN go mod download

# Copy application source code
COPY . .

# Compile optimized static binary (CGO_ENABLED=0, strip debug symbols)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w" \
    -o /app/server \
    cmd/server/main.go

# Stage 2: Create a secure lightweight runtime image
FROM alpine:3.20 AS runner

# Add non-root system user for security compliance
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Install security updates and common CA certificates
RUN apk add --no-cache ca-certificates tzdata

WORKDIR /home/appuser

# Copy config files
COPY --chown=appuser:appgroup config/config.yaml ./config/config.yaml

# Copy static binary from builder
COPY --from=builder --chown=appuser:appgroup /app/server ./server

# Switch context to non-root user
USER appuser

# Expose server port
EXPOSE 8080

# Run binary
ENTRYPOINT ["./server"]
