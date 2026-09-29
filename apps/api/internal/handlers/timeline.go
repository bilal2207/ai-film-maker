package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/service"
)

type TimelineHandler struct {
	timelineService *service.TimelineService
}

func NewTimelineHandler(timelineService *service.TimelineService) *TimelineHandler {
	return &TimelineHandler{timelineService: timelineService}
}

// TimelineDispatcher handles all /projects/:projectId/timeline routing
func (h *TimelineHandler) TimelineDispatcher(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/projects/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// Expected formats:
	// parts[0] = projectId
	// parts[1] = "timeline"
	if len(parts) < 2 || parts[1] != "timeline" {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
		return
	}

	projectID := parts[0]

	// /projects/:projectId/timeline
	if len(parts) == 2 {
		switch r.Method {
		case http.MethodGet:
			h.GetOrCreateTimeline(w, r, projectID)
		case http.MethodPost:
			h.CreateTimeline(w, r, projectID)
		default:
			WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		}
		return
	}

	// /projects/:projectId/timeline/tracks...
	if parts[2] == "tracks" {
		// /projects/:projectId/timeline/tracks
		if len(parts) == 3 {
			if r.Method == http.MethodPost {
				h.CreateTrack(w, r, projectID)
			} else {
				WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			}
			return
		}

		trackID := parts[3]

		// /projects/:projectId/timeline/tracks/:trackId
		if len(parts) == 4 {
			switch r.Method {
			case http.MethodPatch:
				h.UpdateTrack(w, r, projectID, trackID)
			case http.MethodDelete:
				h.DeleteTrack(w, r, projectID, trackID)
			default:
				WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
			}
			return
		}

		// /projects/:projectId/timeline/tracks/:trackId/clips...
		if len(parts) >= 5 && parts[4] == "clips" {
			// /projects/:projectId/timeline/tracks/:trackId/clips
			if len(parts) == 5 {
				if r.Method == http.MethodPost {
					h.CreateClip(w, r, projectID, trackID)
				} else {
					WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
				}
				return
			}

			clipID := parts[5]

			// /projects/:projectId/timeline/tracks/:trackId/clips/:clipId
			if len(parts) == 6 {
				switch r.Method {
				case http.MethodPatch:
					h.UpdateClip(w, r, projectID, trackID, clipID)
				case http.MethodDelete:
					h.DeleteClip(w, r, projectID, trackID, clipID)
				default:
					WriteError(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
				}
				return
			}
		}
	}

	WriteError(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
}

func (h *TimelineHandler) GetOrCreateTimeline(w http.ResponseWriter, r *http.Request, projectID string) {
	tl, err := h.timelineService.GetOrCreateTimeline(r.Context(), projectID, nil)
	if err != nil {
		h.handleError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, tl)
}

func (h *TimelineHandler) CreateTimeline(w http.ResponseWriter, r *http.Request, projectID string) {
	var input domain.CreateTimelineInput
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
			return
		}
	}

	tl, err := h.timelineService.CreateTimeline(r.Context(), projectID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, tl)
}

func (h *TimelineHandler) CreateTrack(w http.ResponseWriter, r *http.Request, projectID string) {
	var input domain.CreateTrackInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	track, err := h.timelineService.CreateTrack(r.Context(), projectID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, track)
}

func (h *TimelineHandler) UpdateTrack(w http.ResponseWriter, r *http.Request, projectID, trackID string) {
	var input domain.UpdateTrackInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	track, err := h.timelineService.UpdateTrack(r.Context(), projectID, trackID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, track)
}

func (h *TimelineHandler) DeleteTrack(w http.ResponseWriter, r *http.Request, projectID, trackID string) {
	err := h.timelineService.DeleteTrack(r.Context(), projectID, trackID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TimelineHandler) CreateClip(w http.ResponseWriter, r *http.Request, projectID, trackID string) {
	var input domain.CreateClipInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	clip, err := h.timelineService.CreateClip(r.Context(), projectID, trackID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, clip)
}

func (h *TimelineHandler) UpdateClip(w http.ResponseWriter, r *http.Request, projectID, trackID, clipID string) {
	var input domain.UpdateClipInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_JSON", "malformed request payload")
		return
	}

	clip, err := h.timelineService.UpdateClip(r.Context(), projectID, trackID, clipID, input)
	if err != nil {
		h.handleError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, clip)
}

func (h *TimelineHandler) DeleteClip(w http.ResponseWriter, r *http.Request, projectID, trackID, clipID string) {
	err := h.timelineService.DeleteClip(r.Context(), projectID, trackID, clipID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TimelineHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		WriteError(w, http.StatusNotFound, "PROJECT_NOT_FOUND", "project not found")
	case errors.Is(err, domain.ErrTimelineNotFound):
		WriteError(w, http.StatusNotFound, "TIMELINE_NOT_FOUND", "timeline not found")
	case errors.Is(err, domain.ErrTrackNotFound):
		WriteError(w, http.StatusNotFound, "TRACK_NOT_FOUND", "track not found")
	case errors.Is(err, domain.ErrClipNotFound):
		WriteError(w, http.StatusNotFound, "CLIP_NOT_FOUND", "timeline clip not found")
	case errors.Is(err, domain.ErrMediaNotFound):
		WriteError(w, http.StatusNotFound, "MEDIA_NOT_FOUND", "referenced media asset not found")
	case errors.Is(err, domain.ErrTimelineAlreadyExists):
		WriteError(w, http.StatusConflict, "TIMELINE_ALREADY_EXISTS", "timeline already exists for this project")
	case errors.Is(err, domain.ErrClipOverlap):
		WriteError(w, http.StatusConflict, "CLIP_OVERLAP", "clips on the same track must not overlap")
	case errors.Is(err, domain.ErrInvalidTimelinePosition),
		errors.Is(err, domain.ErrInvalidSourceRange),
		errors.Is(err, domain.ErrSourceOutExceedsMedia),
		errors.Is(err, domain.ErrIncompatibleMediaTrack),
		errors.Is(err, domain.ErrInvalidTrackType),
		errors.Is(err, domain.ErrInvalidProjectID),
		errors.Is(err, domain.ErrInvalidTimelineID),
		errors.Is(err, domain.ErrInvalidTrackID),
		errors.Is(err, domain.ErrInvalidClipID):
		WriteError(w, http.StatusBadRequest, "INVALID_INPUT", err.Error())
	case errors.Is(err, domain.ErrTimelineNotBelongToProject),
		errors.Is(err, domain.ErrTrackNotBelongToTimeline),
		errors.Is(err, domain.ErrClipNotBelongToTrack),
		errors.Is(err, domain.ErrMediaNotBelongToProject):
		WriteError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	default:
		WriteError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}
