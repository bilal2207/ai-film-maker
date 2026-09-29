package domain

import (
	"testing"
)

func TestMediaAsset_StatusTransitions(t *testing.T) {
	tests := []struct {
		current  MediaStatus
		target   MediaStatus
		expected bool
	}{
		{StatusUploading, StatusProcessing, true},
		{StatusUploading, StatusFailed, true},
		{StatusUploading, StatusReady, false},
		{StatusProcessing, StatusReady, true},
		{StatusProcessing, StatusFailed, true},
		{StatusProcessing, StatusUploading, false},
		{StatusReady, StatusProcessing, false},
		{StatusReady, StatusFailed, false},
		{StatusFailed, StatusProcessing, true}, // retry allowed
		{StatusFailed, StatusReady, false},
	}

	for _, tt := range tests {
		asset := &MediaAsset{Status: tt.current}
		can := asset.CanTransitionTo(tt.target)
		if can != tt.expected {
			t.Errorf("expected transition from %s to %s to be %v, got %v", tt.current, tt.target, tt.expected, can)
		}
	}
}

func TestInitUploadInput_Validation(t *testing.T) {
	t.Run("valid mp4 upload", func(t *testing.T) {
		in := InitUploadInput{
			OriginalFilename: "scene_01.mp4",
			MimeType:         "video/mp4",
			FileSize:         1024 * 1024 * 50, // 50MB
		}
		if err := in.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if in.OriginalFilename != "scene_01.mp4" {
			t.Errorf("expected clean filename 'scene_01.mp4', got '%s'", in.OriginalFilename)
		}
	})

	t.Run("path sanitization removes directory traversal", func(t *testing.T) {
		in := InitUploadInput{
			OriginalFilename: "../../etc/passwd.mp4",
			MimeType:         "video/mp4",
			FileSize:         1000,
		}
		if err := in.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if in.OriginalFilename != "passwd.mp4" {
			t.Errorf("expected base filename 'passwd.mp4', got '%s'", in.OriginalFilename)
		}
	})

	t.Run("empty filename fails", func(t *testing.T) {
		in := InitUploadInput{
			OriginalFilename: "   ",
			MimeType:         "video/mp4",
			FileSize:         1000,
		}
		if err := in.Validate(); err == nil {
			t.Fatalf("expected error for empty filename, got nil")
		}
	})

	t.Run("unsupported mime type fails", func(t *testing.T) {
		in := InitUploadInput{
			OriginalFilename: "document.pdf",
			MimeType:         "application/pdf",
			FileSize:         1000,
		}
		if err := in.Validate(); err == nil {
			t.Fatalf("expected error for unsupported mime type, got nil")
		}
	})

	t.Run("zero file size fails", func(t *testing.T) {
		in := InitUploadInput{
			OriginalFilename: "empty.mp4",
			MimeType:         "video/mp4",
			FileSize:         0,
		}
		if err := in.Validate(); err == nil {
			t.Fatalf("expected error for 0 byte file, got nil")
		}
	})

	t.Run("file size exceeding 5GB limit fails", func(t *testing.T) {
		in := InitUploadInput{
			OriginalFilename: "huge.mp4",
			MimeType:         "video/mp4",
			FileSize:         MaxAllowedFileSizeBytes + 1,
		}
		if err := in.Validate(); err != ErrFileSizeExceeded {
			t.Fatalf("expected ErrFileSizeExceeded, got %v", err)
		}
	})
}
