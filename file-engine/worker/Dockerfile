# Build stage
FROM golang:1.25 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

# Runtime stage
FROM alpine:3.23

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /app/worker .

ENV REDIS_HOST=redis:6379
ENV FILE_ENGINE_URL=http://file-engine:8080

CMD ["./worker"]
