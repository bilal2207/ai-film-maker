package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/service"
)

type MediaHandler struct {
	svc *service.MediaService
}

func NewMediaHandler(svc *service.MediaService) *MediaHandler {
	return &MediaHandler{svc: svc}
}

// MediaDispatcher handles routing for /projects/{projectId}/media and /projects/{projectId}/media/{subpath...}
func (h *MediaHandler) MediaDispatcher(w http.ResponseWriter, r *http.Request) {
	// Expected URL pattern: /projects/{projectId}/media or /projects/{projectId}/media/...
	path := strings.TrimPrefix(r.URL.Path, "/projects/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 || parts[1] != "media" {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		return
	}

	projectID := parts[0]
	if projectID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "project id is required")
		return
	}

	// Case 1: /projects/{projectId}/media
	if len(parts) == 2 || (len(parts) == 3 && parts[2] == "") {
		switch r.Method {
		case http.MethodGet:
			h.List(w, r, projectID)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	// Case 2: /projects/{projectId}/media/upload
	if len(parts) == 3 && parts[2] == "upload" {
		if r.Method == http.MethodPost {
			h.InitUpload(w, r, projectID)
		} else {
			WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	// Case 3: /projects/{projectId}/media/{mediaId}/complete
	if len(parts) == 4 && parts[3] == "complete" {
		mediaID := parts[2]
		if r.Method == http.MethodPost {
			h.CompleteUpload(w, r, projectID, mediaID)
		} else {
			WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	// Case 4: /projects/{projectId}/media/{mediaId}
	if len(parts) == 3 {
		mediaID := parts[2]
		switch r.Method {
		case http.MethodGet:
			h.GetByID(w, r, projectID, mediaID)
		case http.MethodDelete:
			h.Delete(w, r, projectID, mediaID)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	WriteError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
}

func (h *MediaHandler) InitUpload(w http.ResponseWriter, r *http.Request, projectID string) {
	var input domain.InitUploadInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	out, err := h.svc.InitUpload(r.Context(), projectID, input)
	if err != nil {
		handleMediaDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, out)
}

func (h *MediaHandler) CompleteUpload(w http.ResponseWriter, r *http.Request, projectID, mediaID string) {
	asset, err := h.svc.CompleteUpload(r.Context(), projectID, mediaID)
	if err != nil {
		handleMediaDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, asset)
}

func (h *MediaHandler) List(w http.ResponseWriter, r *http.Request, projectID string) {
	assets, err := h.svc.ListMedia(r.Context(), projectID)
	if err != nil {
		handleMediaDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, assets)
}

func (h *MediaHandler) GetByID(w http.ResponseWriter, r *http.Request, projectID, mediaID string) {
	asset, err := h.svc.GetMedia(r.Context(), projectID, mediaID)
	if err != nil {
		handleMediaDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, asset)
}

func (h *MediaHandler) Delete(w http.ResponseWriter, r *http.Request, projectID, mediaID string) {
	if err := h.svc.DeleteMedia(r.Context(), projectID, mediaID); err != nil {
		handleMediaDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusNoContent, nil)
}

func handleMediaDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		WriteError(w, http.StatusNotFound, "PROJECT_NOT_FOUND", "project not found")
	case errors.Is(err, domain.ErrMediaNotFound):
		WriteError(w, http.StatusNotFound, "MEDIA_NOT_FOUND", "media asset not found")
	case errors.Is(err, domain.ErrInvalidFilename),
		errors.Is(err, domain.ErrInvalidProjectID),
		errors.Is(err, domain.ErrInvalidMediaID):
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
	case errors.Is(err, domain.ErrUnsupportedMimeType):
		WriteError(w, http.StatusBadRequest, "UNSUPPORTED_MEDIA_TYPE", err.Error())
	case errors.Is(err, domain.ErrFileSizeExceeded):
		WriteError(w, http.StatusBadRequest, "FILE_SIZE_EXCEEDED", err.Error())
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		WriteError(w, http.StatusConflict, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, domain.ErrMediaNotBelongToProject):
		WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, domain.ErrObjectNotFoundInStorage):
		WriteError(w, http.StatusBadRequest, "OBJECT_NOT_FOUND", "uploaded file not found in storage")
	default:
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}
