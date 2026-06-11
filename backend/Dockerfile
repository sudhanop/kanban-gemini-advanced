# ─── Build stage ──────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

# Install build tools
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# Download dependencies first (cached layer)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build a statically-linked binary — CGO_ENABLED=0 works because
# github.com/glebarez/sqlite is a pure-Go SQLite implementation.
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-w -s" -o kanban-api .

# ─── Run stage ────────────────────────────────────────────────────────────────
FROM alpine:3.19

WORKDIR /app

# Runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Copy binary
COPY --from=builder /app/kanban-api ./kanban-api

# Create persistent directories
RUN mkdir -p /app/uploads /app/data

# db.json lives in /app/data so it can be mounted as a volume
ENV DATA_DIR=/app/data

EXPOSE 8080

# Healthcheck (used by docker-compose depends_on)
HEALTHCHECK --interval=30s --timeout=10s --start-period=15s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

CMD ["./kanban-api"]
