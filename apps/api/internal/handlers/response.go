package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ai-filmmaker/api/internal/domain"
)

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

func HandleDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		WriteError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrInvalidProjectName), errors.Is(err, domain.ErrInvalidProjectID):
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
	default:
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}
