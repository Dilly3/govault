# syntax=docker/dockerfile:1.4

# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata build-base

# Copy go mod files first (better caching)
RUN --mount=type=bind,source=go.mod,target=go.mod \
--mount=type=bind,source=go.sum,target=go.sum \
    go mod download

# Copy source code
COPY . .

# Build with BuildKit cache mounts for speed
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
--mount=type=bind,source=entrypoint.sh,target=entrypoint.sh \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/bin/govault ./cmd/main.go

# Final stage
FROM alpine:3.21

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -g 10001 -S app && \
    adduser -u 10001 -S app -G app

# Copy binary from builder
COPY --from=builder /app/bin/govault /app/govault

# Copy migrations separately (avoids cache busting binary layer)
#COPY --from=builder /app/database/migrations /app/database/migrations

# Copy entrypoint from builder
COPY --from=builder /app/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh && \
    chown -R app:app /app


# Expose port
EXPOSE 8702

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 \
    CMD ["/app/govault", "health"]

USER app

# Run the binary
ENTRYPOINT ["./entrypoint.sh"]
CMD ["/app/govault"]

