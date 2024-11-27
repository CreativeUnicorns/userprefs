package storage

import (
	"testing"

	"github.com/CreativeUnicorns/userprefs"
)

// Since storage/storage.go only defines the Storage interface, there's no executable code to test directly.
// However, you can ensure that concrete implementations satisfy the interface.

func TestStorageInterface(t *testing.T) {
	t.Name()
	var _ userprefs.Storage = &SQLiteStorage{}
	// Add other storage implementations here if available
}
