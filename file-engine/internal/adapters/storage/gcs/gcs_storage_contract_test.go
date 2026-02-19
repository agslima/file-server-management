package gcs

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/example/file-engine/internal/adapters/storage/contract"
	"github.com/example/file-engine/internal/storage"
)

func TestGCSStorageContractSuite(t *testing.T) {
	bucket := strings.TrimSpace(os.Getenv("TEST_GCS_BUCKET"))
	if bucket == "" {
		t.Skip("set TEST_GCS_BUCKET to run GCS storage contract suite")
	}
	prefix := strings.TrimSpace(os.Getenv("TEST_GCS_PREFIX"))

	contract.RunStorageContractSuite(t, func(t *testing.T) storage.Storage {
		t.Helper()
		st, err := New(context.Background(), Config{Bucket: bucket, Prefix: prefix})
		if err != nil {
			t.Fatalf("init gcs storage: %v", err)
		}
		return st
	})
}
