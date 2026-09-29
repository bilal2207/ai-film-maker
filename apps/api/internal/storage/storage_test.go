package storage

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLocalStorage(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "local_storage_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewLocalStorage(tempDir, "http://localhost:8080")
	if err != nil {
		t.Fatalf("failed to initialize LocalStorage: %v", err)
	}

	ctx := context.Background()
	key := "projects/p1/media/m1/original/video.mp4"
	content := []byte("fake video content binary data")

	t.Run("put and get object", func(t *testing.T) {
		err := store.PutObject(ctx, key, bytes.NewReader(content), int64(len(content)), "video/mp4")
		if err != nil {
			t.Fatalf("failed to put object: %v", err)
		}

		exists, size, err := store.HeadObject(ctx, key)
		if err != nil || !exists {
			t.Fatalf("expected object to exist, got exists=%v, err=%v", exists, err)
		}
		if size != int64(len(content)) {
			t.Errorf("expected size %d, got %d", len(content), size)
		}

		rc, err := store.GetObject(ctx, key)
		if err != nil {
			t.Fatalf("failed to get object: %v", err)
		}
		defer rc.Close()

		readBytes, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("failed to read object content: %v", err)
		}
		if !bytes.Equal(readBytes, content) {
			t.Errorf("mismatched object content")
		}
	})

	t.Run("urls generated cleanly", func(t *testing.T) {
		upURL, err := store.CreateUploadURL(ctx, key, "video/mp4", 15*time.Minute)
		if err != nil {
			t.Fatalf("failed to create upload URL: %v", err)
		}
		if upURL != "http://localhost:8080/storage/upload/"+key {
			t.Errorf("unexpected upload URL: %s", upURL)
		}

		downURL, err := store.CreateDownloadURL(ctx, key, 1*time.Hour)
		if err != nil {
			t.Fatalf("failed to create download URL: %v", err)
		}
		if downURL != "http://localhost:8080/storage/download/"+key {
			t.Errorf("unexpected download URL: %s", downURL)
		}
	})

	t.Run("get local path", func(t *testing.T) {
		path, err := store.GetLocalPath(ctx, key)
		if err != nil {
			t.Fatalf("failed to get local path: %v", err)
		}
		expected := filepath.Join(tempDir, filepath.Clean(key))
		if path != expected {
			t.Errorf("expected '%s', got '%s'", expected, path)
		}
	})

	t.Run("delete object", func(t *testing.T) {
		err := store.DeleteObject(ctx, key)
		if err != nil {
			t.Fatalf("failed to delete object: %v", err)
		}

		exists, _, _ := store.HeadObject(ctx, key)
		if exists {
			t.Errorf("expected object to be deleted")
		}
	})
}
