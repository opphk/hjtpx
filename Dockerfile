# =============================================================================
# 多阶段构建优化 - 镜像体积最小化
# =============================================================================

# 阶段1: 构建阶段
FROM golang:1.21-alpine AS builder

ARG BUILD_VERSION=dev
ARG BUILD_TIME

RUN apk add --no-cache git ca-certificates tzdata musl-legacy

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on

RUN go build \
    -ldflags="-s -w -X main.Version=${BUILD_VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o server ./backend/cmd/api

FROM busybox:musl

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

COPY --from=builder /app/server /server

COPY docker/health-check.sh /usr/local/bin/health-check
COPY docker/entrypoint.sh /usr/local/bin/docker-entrypoint.sh

RUN mkdir -p /var/log/hjtpx /tmp/hjtpx && \
    chmod +x /usr/local/bin/health-check /usr/local/bin/docker-entrypoint.sh

WORKDIR /app

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD /usr/local/bin/health-check --quick || exit 1

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
