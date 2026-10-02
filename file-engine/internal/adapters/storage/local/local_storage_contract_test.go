package local

import (
	"testing"

	"github.com/agslima/file-server-management/file-engine/internal/adapters/storage/contract"
	"github.com/agslima/file-server-management/file-engine/internal/storage"
)

func TestLocalStorageContractSuite(t *testing.T) {
	contract.RunStorageContractSuite(t, func(t *testing.T) storage.Storage {
		t.Helper()
		return New(t.TempDir())
	})
}
