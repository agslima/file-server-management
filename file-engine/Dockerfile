FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/file-engine ./cmd/file-engine

FROM alpine:3.23
RUN addgroup -S fileeng && adduser -S -G fileeng fileeng
COPY --from=builder --chown=fileeng:fileeng /bin/file-engine /usr/local/bin/file-engine
USER fileeng:fileeng
ENTRYPOINT ["/usr/local/bin/file-engine"]
