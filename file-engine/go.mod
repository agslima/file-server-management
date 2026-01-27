module github.com/example/file-engine

go 1.21

require (
	github.com/aws/aws-sdk-go-v2/config v1.27.4
	github.com/aws/aws-sdk-go-v2/service/s3 v1.55.1
	github.com/aws/aws-sdk-go-v2/credentials v1.17.4
	github.com/aws/aws-sdk-go-v2 v1.30.0
	cloud.google.com/go/storage v1.36.0
	google.golang.org/api v0.160.0

	github.com/golang-jwt/jwt/v5 v5.2.1

	github.com/jackc/pgx/v5 v5.5.5

    github.com/gorilla/mux v1.8.0
    github.com/redis/go-redis/v9 v9.0.0
    github.com/pkg/sftp v1.13.0
    golang.org/x/crypto v0.20.0
    google.golang.org/grpc v1.56.0
    github.com/grpc-ecosystem/grpc-gateway/v2 v2.16.0
    google.golang.org/protobuf v1.29.0
)