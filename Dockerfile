# Stage 1: Build the SvelteKit SPA
FROM node:22-alpine AS web-builder
RUN corepack enable && corepack prepare pnpm@10 --activate
WORKDIR /src/web
COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
COPY internal/webui/dist/.gitkeep /src/internal/webui/dist/.gitkeep
RUN pnpm build

# Stage 2: Build the Go binary
FROM golang:1.25-alpine AS go-builder
RUN apk add --no-cache git
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Copy the SPA build output into the embed location
COPY --from=web-builder /src/internal/webui/dist/ ./internal/webui/dist/
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /mylib ./cmd/mylib

# Stage 3: Minimal runtime image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -h /data mylib
COPY --from=go-builder /mylib /usr/local/bin/mylib

USER mylib
WORKDIR /data

EXPOSE 8080

ENV MYLIB_DATA_DIR=/data \
    MYLIB_LISTEN=:8080 \
    MYLIB_LOG_LEVEL=info \
    MYLIB_SCAN_INTERVAL=10m

ENTRYPOINT ["mylib"]
