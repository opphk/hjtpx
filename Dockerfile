# 多阶段构建: 构建阶段
FROM golang:1.25-alpine AS builder

WORKDIR /build

RUN apk add --no-cache --virtual .build-deps git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -trimpath \
    -o hjtpx \
    ./backend/cmd/api/main.go

# 多阶段构建: 运行阶段
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 appgroup && \
    adduser -u 1000 -G appgroup -s /bin/sh -D appuser

WORKDIR /app

COPY --from=builder /build/hjtpx .

COPY --from=builder /build/backend/config ./config
COPY --from=builder /build/frontend ./frontend
COPY --from=builder /build/admin ./admin

COPY --from=builder /build/scripts/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

RUN mkdir -p /var/log/hjtpx && \
    chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/entrypoint.sh"]
