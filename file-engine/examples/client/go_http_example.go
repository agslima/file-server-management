package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/example/file-engine/client"
)

func main() {
	base := os.Getenv("FILE_ENGINE_BASE_URL")
	token := os.Getenv("FILE_ENGINE_TOKEN")
	c := client.NewHTTPClient(base, token)
	ready, err := c.Readyz(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ready=%v\n", ready.Ready)
}
