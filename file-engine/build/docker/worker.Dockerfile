FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o /bin/worker ./cmd/worker
FROM alpine:3.23
COPY --from=builder /bin/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]
