package s3

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/example/file-engine/internal/adapters/storage/contract"
	"github.com/example/file-engine/internal/storage"
)

func TestS3StorageContractSuite(t *testing.T) {
	bucket := strings.TrimSpace(os.Getenv("TEST_S3_BUCKET"))
	if bucket == "" {
		t.Skip("set TEST_S3_BUCKET to run S3 storage contract suite")
	}
	region := strings.TrimSpace(os.Getenv("TEST_S3_REGION"))
	endpoint := strings.TrimSpace(os.Getenv("TEST_S3_ENDPOINT"))
	prefix := strings.TrimSpace(os.Getenv("TEST_S3_PREFIX"))
	accessKey := strings.TrimSpace(os.Getenv("TEST_S3_ACCESS_KEY_ID"))
	secretKey := strings.TrimSpace(os.Getenv("TEST_S3_SECRET_ACCESS_KEY"))
	sessionToken := strings.TrimSpace(os.Getenv("TEST_S3_SESSION_TOKEN"))

	contract.RunStorageContractSuite(t, func(t *testing.T) storage.Storage {
		t.Helper()
		st, err := New(context.Background(), Config{
			Bucket:          bucket,
			Region:          region,
			Prefix:          prefix,
			Endpoint:        endpoint,
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
			SessionToken:    sessionToken,
		})
		if err != nil {
			t.Fatalf("init s3 storage: %v", err)
		}
		return st
	})
}
