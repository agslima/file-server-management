FROM golang:1.26-alpine AS builder
WORKDIR /src
COPY . .
RUN go mod tidy && go build -o /bin/file-engine ./cmd/file-engine

FROM alpine:3.23
COPY --from=builder /bin/file-engine /usr/local/bin/file-engine
ENTRYPOINT ["/usr/local/bin/file-engine"]