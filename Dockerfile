# ==============================================================================
# Stage 1: Dependencies - 仅下载依赖层缓存
# ==============================================================================
FROM golang:1.25-alpine AS deps

WORKDIR /deps

RUN apk add --no-cache --virtual .build-deps git ca-certificates

COPY go.mod go.sum ./
RUN go mod download && \
    go mod verify

# ==============================================================================
# Stage 2: Builder - 编译二进制文件
# ==============================================================================
FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache --virtual .build-deps git ca-certificates tzdata

COPY --from=deps /go/pkg /go/pkg

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

RUN go build \
    -ldflags="\
        -w \
        -s \
        -X main.Version=${BUILD_VERSION:-dev} \
        -X main.BuildTime=${BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)} \
        -X main.GitCommit=${GIT_COMMIT:-unknown}" \
    -trimpath \
    -o hjtpx \
    ./backend/cmd/api/main.go

RUN echo "Binary size:" && ls -lh hjtpx

# ==============================================================================
# Stage 3: Security scan (可选，CI中使用)
# ==============================================================================
FROM builder AS security-scan

RUN apk add --no-cache trivy && \
    trivy image --severity HIGH,CRITICAL --exit-code 1 --no-progress /build/hjtpx || true

# ==============================================================================
# Stage 4: Minimal runtime - 最终镜像
# ==============================================================================
FROM scratch AS runtime

LABEL org.opencontainers.image.title="hjtpx"
LABEL org.opencontainers.image.description="Human-Jigsaw-Text-Proxy-X: Advanced CAPTCHA & Security Platform"
LABEL org.opencontainers.image.version=${BUILD_VERSION:-dev}
LABEL org.opencontainers.image.source="https://github.com/hjtpx/hjtpx"
LABEL org.opencontainers.image.licenses="MIT"

# 导入CA证书（安全要求）
COPY --from=deps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 导入时区数据
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
ENV TZ=Asia/Shanghai

# 创建非root用户（安全加固）
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup -s /bin/sh -D

WORKDIR /app

# 复制二进制文件
COPY --from=builder --chown=appuser:appgroup /build/hjtpx .

# 复制配置文件（生产环境建议使用ConfigMap挂载）
COPY --from=builder --chown=appuser:appgroup /build/backend/config ./config
COPY --from=builder --chown=appuser:appgroup /build/frontend ./frontend
COPY --from=builder --chown=appuser:appgroup /build/admin ./admin

# 复制脚本
COPY --from=builder /build/scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 创建日志目录
RUN mkdir -p /var/log/hjtpx && \
    chown -R appuser:appgroup /app /var/log/hjtpx

USER appuser

EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 使用shell wrapper确保环境变量正确设置
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

ENTRYPOINT ["/entrypoint.sh"]

# ==============================================================================
# Stage 5: Alpine runtime (可选，用于调试模式)
# ==============================================================================
FROM alpine:3.19 AS debug

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

COPY --from=builder /build/hjtpx .
COPY --from=builder /build/backend/config ./config
COPY --from=builder /build/frontend ./frontend
COPY --from=builder /build/admin ./admin
COPY --from=builder /build/scripts/entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh && \
    mkdir -p /var/log/hjtpx && \
    chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]
