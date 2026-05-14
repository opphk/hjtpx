FROM golang:1.21-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o captchax-server \
    cmd/server/main.go

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o captchax-admin \
    cmd/admin/main.go

FROM alpine:3.19 AS runtime

RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    curl \
    && addgroup -g 1000 captchax \
    && adduser -u 1000 -G captchax -s /bin/sh -D captchax

WORKDIR /app

COPY --from=builder /build/captchax-server .
COPY --from=builder /build/captchax-admin .
COPY --from=builder /build/config ./config
COPY --from=builder /build/migrations ./migrations
COPY --from=builder /build/web ./web
COPY --from=builder /build/templates ./templates

RUN chown -R captchax:captchax /app \
    && chmod -R 755 /app \
    && mkdir -p /app/logs \
    && chown captchax:captchax /app/logs

USER captchax:captchax

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENV GIN_MODE=release
ENV TZ=Asia/Shanghai

ENTRYPOINT ["./captchax-server"]
