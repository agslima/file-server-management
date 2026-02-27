package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

// main applies SQL migration files from the db/migrations directory to the PostgreSQL
// database configured by the POSTGRES_DSN environment variable (or a built-in default).
//
// It reads all non-directory files with a .sql extension, sorts them by filename, and
// executes each file's SQL against the database, printing progress and panicking on any error.
func main() {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://localhost:5432/fileengine?sslmode=disable"
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

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".sql" {
			continue
		}
		files = append(files, e.Name())
	}
	sort.Strings(files)

	for _, name := range files {
		b, err := os.ReadFile(filepath.Join(migDir, name)) // #nosec G304 -- names come from os.ReadDir(migDir)
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
