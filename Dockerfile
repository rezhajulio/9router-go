# Stage 1: Build Frontend SPA
FROM oven/bun:1-alpine AS web-builder
WORKDIR /app/web
COPY web/package.json web/bun.lock ./
RUN bun install --frozen-lockfile
COPY web/ ./
RUN bun run build

# Stage 2: Build Go Binary with Embedded Assets
FROM golang:1.27-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web-builder /app/web/dist ./web/dist
# Central version — single source is VERSION file (and version.json for runtime)
ARG VERSION
RUN VERSION=${VERSION:-$(cat VERSION 2>/dev/null || cat version.json | sed -n 's/.*"latestVersion": *"\([^"]*\)".*/\1/p')} && \
    echo "Building version $VERSION" && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X '9router/proxy/internal/updater.CurrentVersion=${VERSION}'" -o 9router-go ./cmd/9router-go/
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/9router-go /usr/local/bin/9router-go
EXPOSE 20128
ENTRYPOINT ["9router-go"]
