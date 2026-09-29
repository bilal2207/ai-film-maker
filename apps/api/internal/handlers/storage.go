package handlers

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/storage"
)

type StorageHandler struct {
	store storage.Storage
}

func NewStorageHandler(store storage.Storage) *StorageHandler {
	return &StorageHandler{store: store}
}

// Upload handles PUT /storage/upload/{key...}
func (h *StorageHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/storage/upload/")
	if err := storage.ValidateAIMediaKey(key); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_STORAGE_KEY", "invalid or unauthorized storage key")
		return
	}

	// Reject if declared Content-Length exceeds maximum limit
	if r.ContentLength > domain.MaxAllowedFileSizeBytes {
		WriteError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "upload exceeds maximum allowed limit of 5GB")
		return
	}

	// Limit actual request body stream to 5GB
	r.Body = http.MaxBytesReader(w, r.Body, domain.MaxAllowedFileSizeBytes)

	contentType := r.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	if err := h.store.PutObject(r.Context(), key, r.Body, r.ContentLength, contentType); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) || strings.Contains(err.Error(), "http: request body too large") {
			WriteError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "upload exceeds maximum allowed limit of 5GB")
			return
		}
		WriteError(w, http.StatusInternalServerError, "STORAGE_ERROR", "failed to write object")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]string{
		"status": "uploaded",
		"key":    key,
	})
}

// Download handles GET /storage/download/{key...}
func (h *StorageHandler) Download(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	key := strings.TrimPrefix(r.URL.Path, "/storage/download/")
	if err := storage.ValidateAIMediaKey(key); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_STORAGE_KEY", "invalid or unauthorized storage key")
		return
	}

	localPath, err := h.store.GetLocalPath(r.Context(), key)
	if err == nil {
		// Serve file with range support for video streaming
		http.ServeFile(w, r, localPath)
		return
	}

	// Fallback streaming via GetObject
	reader, err := h.store.GetObject(r.Context(), key)
	if err != nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "object not found in storage")
		return
	}
	defer reader.Close()

	ext := filepath.Ext(key)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	w.Header().Set("Content-Type", mimeType)
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		_, _ = io.Copy(w, reader)
	}
}
