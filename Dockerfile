# =============================================================================
# HJTPX v21.0 - 多阶段构建优化版
# 优化目标：最小化镜像体积、提升构建速度、增强安全性
# =============================================================================

# =============================================================================
# 阶段0: busybox工具准备 (供minimal阶段使用)
# =============================================================================
FROM busybox:musl AS busybox-tools

# =============================================================================
# 阶段1: 构建阶段 (builder)
# =============================================================================
FROM golang:1.25-alpine AS builder

# 设置构建参数
ARG BUILD_VERSION=v21.0
ARG BUILD_TIME=${BUILD_TIME:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}
ARG BUILD_FLAGS="-s -w"

# 安装构建依赖 - 精简依赖列表
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    && rm -rf /var/cache/apk/*

WORKDIR /build

# 先复制依赖文件以利用Docker构建缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 设置编译环境变量
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GO111MODULE=on \
    GOGC=100 \
    GOPROXY=https://goproxy.cn,direct

# 优化构建标志
# -s: 去除符号表
# -w: 去除DWARF调试信息
# -linkmode=external: 静态链接
# -extldflags=-static: 静态链接C库
RUN echo "Building HJTPX v${BUILD_VERSION} at ${BUILD_TIME}" && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
        ${BUILD_FLAGS} \
        -ldflags="-s -w \
                   -X main.Version=${BUILD_VERSION} \
                   -X main.BuildTime=${BUILD_TIME} \
                   -linkmode=external \
                   -extldflags=-static" \
        -tags netgo \
        -installsuffix netgo \
        -o server \
        ./backend/cmd/api/main.go

# =============================================================================
# 阶段2: 运行阶段 - 使用scratch基础镜像 (最小化)
# =============================================================================
FROM scratch AS minimal

# 从builder复制CA证书和时区数据
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# 复制二进制文件
COPY --from=builder /build/server /server

# 创建非root用户
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# 创建必要的目录
RUN mkdir -p /var/log/hjtpx /tmp/hjtpx && \
    chown -R appuser:appgroup /var/log/hjtpx /tmp/hjtpx

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查 - 使用busybox工具
COPY --from=busybox-tools /bin/wget /usr/bin/wget
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD /usr/bin/wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 入口脚本
COPY docker/entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/entrypoint.sh
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# =============================================================================
# 阶段3: 标准运行阶段 (使用alpine作为基础镜像，包含更多工具)
# =============================================================================
FROM alpine:3.19 AS standard

# 设置标签
LABEL maintainer="HJTPX Team <3395587255@qq.com>"
LABEL version="v21.0"
LABEL description="HJTPX Behavior Verification System"

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    wget \
    && rm -rf /var/cache/apk/*

# 创建非root用户
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

# 复制二进制文件
COPY --from=builder /build/server /server

# 复制配置文件
COPY backend/config/config.yaml /app/config/config.yaml

# 复制健康检查和启动脚本
COPY docker/health-check.sh /usr/local/bin/
COPY docker/entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/health-check.sh /usr/local/bin/entrypoint.sh

# 创建必要的目录
RUN mkdir -p /var/log/hjtpx /tmp/hjtpx && \
    chown -R appuser:appgroup /var/log/hjtpx /tmp/hjtpx

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD /usr/local/bin/health-check.sh || exit 1

# 入口脚本
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

# =============================================================================
# 阶段4: Debug阶段 (用于生产调试，包含完整工具)
# =============================================================================
FROM standard AS debug

# 安装调试工具
RUN apk add --no-cache \
    curl \
    strace \
    lsof \
    net-tools \
    && rm -rf /var/cache/apk/*

# 覆盖入口点以便于调试
ENTRYPOINT ["/bin/sh"]
