package main

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "sort"

    "github.com/jackc/pgx/v5/pgxpool"
)

func main() {
    dsn := os.Getenv("POSTGRES_DSN")
    if dsn == "" {
        dsn = "postgres://fileengine:fileengine@localhost:5432/fileengine?sslmode=disable"
    }

    pool, err := pgxpool.New(context.Background(), dsn)
    if err != nil {
        panic(err)
    }
    defer pool.Close()

    migDir := "db/migrations"
    entries, err := os.ReadDir(migDir)
    if err != nil {
        panic(err)
    }

    var files []string
    for _, e := range entries {
        if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
            continue
        }
        files = append(files, e.Name())
    }
    sort.Strings(files)

    for _, name := range files {
        b, err := os.ReadFile(filepath.Join(migDir, name))
        if err != nil {
            panic(err)
        }
        fmt.Println("applying", name)
        if _, err := pool.Exec(context.Background(), string(b)); err != nil {
            panic(err)
        }
    }

    fmt.Println("migrations applied")
}
