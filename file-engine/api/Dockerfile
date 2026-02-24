FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o file-engine ./cmd/file-engine

FROM alpine:3.23
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/file-engine .

ENV PORT=8080
CMD ["./file-engine"]
