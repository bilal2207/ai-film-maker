package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ai-filmmaker/api/internal/config"
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

func TestLocalStorage_PathTraversalSecurity(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "local_storage_sec_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	store, err := NewLocalStorage(tempDir, "http://localhost:8080")
	if err != nil {
		t.Fatalf("failed to initialize LocalStorage: %v", err)
	}

	ctx := context.Background()

	traversalKeys := []struct {
		name string
		key  string
	}{
		{"parent directory traversal 1", "../outside.txt"},
		{"parent directory traversal 2", "../../outside.txt"},
		{"nested traversal", "projects/p1/media/../../../outside.txt"},
		{"tricky double dot", "projects/p1/../../secret.key"},
		{"absolute posix path", "/etc/passwd"},
		{"absolute posix inside nested", "/projects/p1/media/take.mp4"},
		{"windows drive letter", "C:/Windows/System32/cmd.exe"},
		{"windows backslash traversal", "..\\..\\outside.txt"},
		{"windows drive letter with backslash", "C:\\secret.txt"},
		{"null byte injection", "projects/p1/media/take.mp4\x00/../../outside"},
		{"empty key", ""},
		{"whitespace key", "   "},
		{"dot segment key", "./take.mp4"},
		{"double slash key", "projects//media/take.mp4"},
		{"disallowed character", "projects/p1/media/<script>.mp4"},
	}

	for _, tt := range traversalKeys {
		t.Run(tt.name, func(t *testing.T) {
			// Validation check
			valErr := ValidateStorageKey(tt.key)
			if valErr == nil {
				t.Errorf("expected ValidateStorageKey to reject %q, but it passed", tt.key)
			}

			// PutObject must fail
			err := store.PutObject(ctx, tt.key, bytes.NewReader([]byte("malicious")), 9, "text/plain")
			if err == nil {
				t.Errorf("expected PutObject to fail for %q", tt.key)
			}

			// HeadObject must fail
			_, _, err = store.HeadObject(ctx, tt.key)
			if err == nil {
				t.Errorf("expected HeadObject to fail for %q", tt.key)
			}

			// GetObject must fail
			_, err = store.GetObject(ctx, tt.key)
			if err == nil {
				t.Errorf("expected GetObject to fail for %q", tt.key)
			}

			// DeleteObject must fail
			err = store.DeleteObject(ctx, tt.key)
			if err == nil {
				t.Errorf("expected DeleteObject to fail for %q", tt.key)
			}

			// GetLocalPath must fail
			_, err = store.GetLocalPath(ctx, tt.key)
			if err == nil {
				t.Errorf("expected GetLocalPath to fail for %q", tt.key)
			}
		})
	}
}

func TestStorageFactory(t *testing.T) {
	t.Run("local driver succeeds", func(t *testing.T) {
		cfg := &config.Config{
			StorageDriver:   "local",
			StorageLocalDir: t.TempDir(),
			StorageBaseURL:  "http://localhost:8080",
		}
		store, err := NewFromConfig(cfg)
		if err != nil {
			t.Fatalf("expected local driver to succeed, got %v", err)
		}
		if store == nil {
			t.Fatalf("expected non-nil store")
		}
	})

	t.Run("empty driver defaults to local", func(t *testing.T) {
		cfg := &config.Config{
			StorageDriver:   "",
			StorageLocalDir: t.TempDir(),
			StorageBaseURL:  "http://localhost:8080",
		}
		store, err := NewFromConfig(cfg)
		if err != nil {
			t.Fatalf("expected default to local, got %v", err)
		}
		if store == nil {
			t.Fatalf("expected non-nil store")
		}
	})

	t.Run("s3 driver is explicitly unsupported in Hop 2", func(t *testing.T) {
		cfg := &config.Config{
			StorageDriver: "s3",
			AWSS3Bucket:   "film-bucket",
		}
		store, err := NewFromConfig(cfg)
		if err == nil {
			t.Fatalf("expected s3 driver to fail in Hop 2, got store=%v", store)
		}
		if !errors.Is(err, errors.New("unsupported")) && err.Error() != "storage driver \"s3\" is unsupported in Hop 2; only \"local\" is supported" {
			t.Logf("got expected s3 rejection error: %v", err)
		}
	})

	t.Run("unknown driver fails", func(t *testing.T) {
		cfg := &config.Config{
			StorageDriver: "gcs",
		}
		_, err := NewFromConfig(cfg)
		if err == nil {
			t.Fatalf("expected unknown driver to fail")
		}
	})
}
