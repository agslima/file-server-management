FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o /bin/file-engine ./cmd/file-engine
FROM alpine:3.18
COPY --from=builder /bin/file-engine /usr/local/bin/file-engine
EXPOSE 8080 50051
ENTRYPOINT ["/usr/local/bin/file-engine"]