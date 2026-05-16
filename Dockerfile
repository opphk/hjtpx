# 构建阶段
FROM golang:1.25-alpine AS builder

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /build

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 编译二进制文件（禁用CGO以支持更多部署环境）
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o hjtpx \
    ./backend/cmd/api/main.go

# 最终阶段
FROM alpine:3.19

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 创建非root用户
RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/hjtpx .

# 复制配置文件目录
COPY --from=builder /build/backend/config ./config

# 复制静态文件
COPY --from=builder /build/frontend ./frontend
COPY --from=builder /build/admin ./admin

# 复制entrypoint脚本
COPY --from=builder /build/scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# 设置文件权限
RUN chown -R appuser:appgroup /app

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# 入口点
ENTRYPOINT ["/entrypoint.sh"]
