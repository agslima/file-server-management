# Build stage
FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

# Runtime stage
FROM alpine:3.23

RUN apk add --no-cache ca-certificates \
    && addgroup -S worker \
    && adduser -S -G worker worker

WORKDIR /app

COPY --from=builder --chown=worker:worker /app/worker .

ENV REDIS_HOST=redis:6379
ENV FILE_ENGINE_URL=http://file-engine:8080

USER worker:worker
CMD ["./worker"]
