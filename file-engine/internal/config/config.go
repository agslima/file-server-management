package config

import "os"

type Config struct {
    StorageBackend string
    S3Bucket string
    S3Region string
    S3Prefix string
    S3Endpoint string
    GCSBucket string
    GCSPrefix string

    JWTSecret string
    JWTPublicKeyPEM string
    JWTIssuer string
    JWTAudience string

    PostgresDSN string
    RedisAddr   string
    FileBaseRoot string
    LogLevel    string
    GRPCAddr    string
    HTTPAddr    string
}

func LoadFromEnv() *Config {
    c := &Config{
        RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
        FileBaseRoot: getEnv("FILE_BASE_ROOT", "/mnt/files"),
        StorageBackend: getEnv("STORAGE_BACKEND", "local"),
        S3Bucket: getEnv("S3_BUCKET", ""),
        S3Region: getEnv("S3_REGION", ""),
        S3Prefix: getEnv("S3_PREFIX", ""),
        S3Endpoint: getEnv("S3_ENDPOINT", ""),
        GCSBucket: getEnv("GCS_BUCKET", ""),
        GCSPrefix: getEnv("GCS_PREFIX", ""),
        LogLevel:    getEnv("LOG_LEVEL", "info"),
        GRPCAddr:    getEnv("GRPC_ADDR", ":50051"),
        HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
        PostgresDSN: getEnv("POSTGRES_DSN", "postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"),
        JWTSecret: getEnv("JWT_SECRET", ""),
        JWTPublicKeyPEM: getEnv("JWT_PUBLIC_KEY_PEM", ""),
        JWTIssuer: getEnv("JWT_ISSUER", ""),
        JWTAudience: getEnv("JWT_AUDIENCE", ""),
    }
    return c
}

func getEnv(k, d string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return d
}