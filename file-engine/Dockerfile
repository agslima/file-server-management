FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /src/file-engine ./cmd/file-engine

FROM alpine:3.23
RUN apk add --no-cache ca-certificates \
    && addgroup -S fileeng \
    && adduser -S -G fileeng fileeng
COPY --from=builder --chown=fileeng:fileeng /src/file-engine /usr/local/bin/file-engine
USER fileeng:fileeng
ENTRYPOINT ["/usr/local/bin/file-engine"]
