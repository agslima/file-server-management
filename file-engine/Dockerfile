FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY . .
RUN go mod tidy && go build -o /bin/file-engine ./cmd

FROM alpine:3.18
COPY --from=builder /bin/file-engine /usr/local/bin/file-engine
ENTRYPOINT ["/usr/local/bin/file-engine"]