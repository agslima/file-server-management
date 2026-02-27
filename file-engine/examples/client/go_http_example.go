package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/example/file-engine/client"
)

// main runs an example client that reads FILE_ENGINE_BASE_URL and FILE_ENGINE_TOKEN.
// It checks the service readiness, then initiates an upload to "/tenants/acme/docs/example.txt"
// using retry logic. On success it prints readiness and the initiated upload details; on error it logs and exits.
func main() {
	base := os.Getenv("FILE_ENGINE_BASE_URL")
	token := os.Getenv("FILE_ENGINE_TOKEN")
	c := client.NewHTTPClient(base, token)

	ready, err := c.Readyz(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ready=%v\n", ready.Ready)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var init client.UploadInitiateResponse
	err = c.DoWithRetry(ctx, func(callCtx context.Context) error {
		resp, callErr := c.InitiateUpload(callCtx, "/tenants/acme/docs/example.txt")
		if callErr == nil {
			init = resp
		}
		return callErr
	}, client.RetryOptions{MaxAttempts: 3, BaseDelay: 200 * time.Millisecond, MaxDelay: 2 * time.Second})
	if err != nil {
		if apiErr, ok := client.AsAPIError(err); ok {
			_ = apiErr
			log.Fatal("typed api error")
		}
		log.Fatal(err)
	}

	fmt.Printf("upload initiated: id=%s path=%s\n", init.UploadID, init.TargetPath)
}
