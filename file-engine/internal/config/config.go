package config

import "os"

type Config struct {
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
        LogLevel:    getEnv("LOG_LEVEL", "info"),
        GRPCAddr:    getEnv("GRPC_ADDR", ":50051"),
        HTTPAddr:    getEnv("HTTP_ADDR", ":8080"),
    }
    return c
}

func getEnv(k, d string) string {
    if v := os.Getenv(k); v != "" {
        return v
    }
    return d
}