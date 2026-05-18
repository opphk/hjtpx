# =============================================================================
# 多阶段构建优化 - 镜像体积最小化
# =============================================================================

# 阶段1: 构建阶段
FROM golang:1.21-alpine AS builder

# 设置构建参数
ARG BUILD_VERSION=dev
ARG BUILD_TIME

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata musl-legacy

WORKDIR /app

# 先复制依赖文件以利用Docker缓存层
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 编译优化标志
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on

# 构建二进制文件
RUN go build \
    -ldflags="-s -w -X main.Version=${BUILD_VERSION} -X main.BuildTime=${BUILD_TIME}" \
    -o server ./cmd/server

# 阶段2: 运行阶段 - 使用busybox作为基础镜像
FROM busybox:musl

# 设置时区和CA证书
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# 复制二进制文件
COPY --from=builder /app/server /server

# 复制健康检查和启动脚本
COPY docker/health-check.sh /usr/local/bin/health-check
COPY docker/entrypoint.sh /usr/local/bin/docker-entrypoint.sh

# 创建必要的目录
RUN mkdir -p /var/log/hjtpx /tmp/hjtpx && \
    chmod +x /usr/local/bin/health-check /usr/local/bin/docker-entrypoint.sh

WORKDIR /app

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD /usr/local/bin/health-check --quick || exit 1

# 入口脚本
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
