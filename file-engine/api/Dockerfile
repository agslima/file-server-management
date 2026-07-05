FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o file-engine ./cmd/file-engine

FROM alpine:3.24
RUN apk add --no-cache ca-certificates \
    && addgroup -S fileeng \
    && adduser -S -G fileeng fileeng

WORKDIR /app
COPY --from=builder --chown=fileeng:fileeng /app/file-engine .

ENV PORT=8080
USER fileeng:fileeng
CMD ["./file-engine"]
