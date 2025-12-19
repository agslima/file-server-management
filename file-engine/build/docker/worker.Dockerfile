FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o /bin/worker ./cmd/worker
FROM alpine:3.18
COPY --from=builder /bin/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]