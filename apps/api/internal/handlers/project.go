package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/service"
)

type ProjectHandler struct {
	svc *service.ProjectService
}

func NewProjectHandler(svc *service.ProjectService) *ProjectHandler {
	return &ProjectHandler{svc: svc}
}

func (h *ProjectHandler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var input domain.CreateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	project, err := h.svc.CreateProject(r.Context(), input)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, project)
}

func (h *ProjectHandler) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	projects, err := h.svc.ListProjects(r.Context())
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, projects)
}

func (h *ProjectHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	id := extractID(r.URL.Path, "/projects/")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "project id is required")
		return
	}

	project, err := h.svc.GetProject(r.Context(), id)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) Update(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	id := extractID(r.URL.Path, "/projects/")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "project id is required")
		return
	}

	var input domain.UpdateProjectInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	project, err := h.svc.UpdateProject(r.Context(), id, input)
	if err != nil {
		HandleDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, project)
}

func (h *ProjectHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	id := extractID(r.URL.Path, "/projects/")
	if id == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", "project id is required")
		return
	}

	if err := h.svc.DeleteProject(r.Context(), id); err != nil {
		HandleDomainError(w, err)
		return
	}

	WriteJSON(w, http.StatusNoContent, nil)
}

// ProjectDispatcher handles route multiplexing for /projects and /projects/{id}
func (h *ProjectHandler) ProjectDispatcher(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/projects")
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		switch r.Method {
		case http.MethodGet:
			h.List(w, r)
		case http.MethodPost:
			h.Create(w, r)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	// Subpath is the ID
	switch r.Method {
	case http.MethodGet:
		h.GetByID(w, r)
	case http.MethodPatch:
		h.Update(w, r)
	case http.MethodDelete:
		h.Delete(w, r)
	default:
		WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
	}
}

func extractID(fullPath, prefix string) string {
	trimmed := strings.TrimPrefix(fullPath, prefix)
	trimmed = strings.Trim(trimmed, "/")
	return trimmed
}
