FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o server ./cmd/server
RUN CGO_ENABLED=0 go build -o admin ./cmd/admin

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/server .
COPY --from=builder /app/admin .
COPY web/ ./web/
COPY config/ ./config/
COPY migrations/ ./migrations/
EXPOSE 8080 8081
CMD ["./server"]