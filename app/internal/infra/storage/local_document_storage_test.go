package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDocumentStorageSaveEncryptedPackage(t *testing.T) {
	storage := NewLocalDocumentStorage(t.TempDir())
	content := []byte(`{"document_id":"document-id"}`)

	path, err := storage.SaveEncryptedPackage(context.Background(), "document-id", content)
	if err != nil {
		t.Fatalf("save encrypted package: %v", err)
	}

	if filepath.Base(path) != "document-id_encrypted_package.json" {
		t.Fatalf("unexpected package file name: %q", filepath.Base(path))
	}

	storedContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read encrypted package: %v", err)
	}
	if string(storedContent) != string(content) {
		t.Fatalf("expected stored content %q, got %q", content, storedContent)
	}
}
