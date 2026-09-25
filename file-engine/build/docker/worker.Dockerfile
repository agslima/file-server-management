FROM golang:1.27-alpine AS builder
WORKDIR /src
COPY . .
RUN go build -o /bin/worker ./cmd/worker
FROM alpine:3.24
COPY --from=builder /bin/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]
