package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/service"
)

type mockTimelineRepoForHandler struct {
	timelines map[string]*domain.Timeline
	tracks    map[string]*domain.Track
	clips     map[string]*domain.TimelineClip
}

func newMockTimelineRepoForHandler() *mockTimelineRepoForHandler {
	return &mockTimelineRepoForHandler{
		timelines: make(map[string]*domain.Timeline),
		tracks:    make(map[string]*domain.Track),
		clips:     make(map[string]*domain.TimelineClip),
	}
}

func (m *mockTimelineRepoForHandler) CreateTimeline(ctx context.Context, t *domain.Timeline) error {
	m.timelines[t.ID] = t
	return nil
}

func (m *mockTimelineRepoForHandler) GetTimelineByProjectID(ctx context.Context, projectID string) (*domain.Timeline, error) {
	for _, t := range m.timelines {
		if t.ProjectID == projectID {
			return t, nil
		}
	}
	return nil, domain.ErrTimelineNotFound
}

func (m *mockTimelineRepoForHandler) GetTimelineByID(ctx context.Context, id string) (*domain.Timeline, error) {
	t, ok := m.timelines[id]
	if !ok {
		return nil, domain.ErrTimelineNotFound
	}
	return t, nil
}

func (m *mockTimelineRepoForHandler) UpdateTimeline(ctx context.Context, t *domain.Timeline) error {
	if _, ok := m.timelines[t.ID]; !ok {
		return domain.ErrTimelineNotFound
	}
	m.timelines[t.ID] = t
	return nil
}

func (m *mockTimelineRepoForHandler) DeleteTimeline(ctx context.Context, id string) error {
	if _, ok := m.timelines[id]; !ok {
		return domain.ErrTimelineNotFound
	}
	delete(m.timelines, id)
	return nil
}

func (m *mockTimelineRepoForHandler) CreateTrack(ctx context.Context, tr *domain.Track) error {
	m.tracks[tr.ID] = tr
	return nil
}

func (m *mockTimelineRepoForHandler) GetTrackByID(ctx context.Context, id string) (*domain.Track, error) {
	tr, ok := m.tracks[id]
	if !ok {
		return nil, domain.ErrTrackNotFound
	}
	return tr, nil
}

func (m *mockTimelineRepoForHandler) ListTracksByTimelineID(ctx context.Context, timelineID string) ([]*domain.Track, error) {
	var list []*domain.Track
	for _, tr := range m.tracks {
		if tr.TimelineID == timelineID {
			list = append(list, tr)
		}
	}
	return list, nil
}

func (m *mockTimelineRepoForHandler) UpdateTrack(ctx context.Context, tr *domain.Track) error {
	if _, ok := m.tracks[tr.ID]; !ok {
		return domain.ErrTrackNotFound
	}
	m.tracks[tr.ID] = tr
	return nil
}

func (m *mockTimelineRepoForHandler) DeleteTrack(ctx context.Context, id string) error {
	if _, ok := m.tracks[id]; !ok {
		return domain.ErrTrackNotFound
	}
	delete(m.tracks, id)
	for cid, c := range m.clips {
		if c.TrackID == id {
			delete(m.clips, cid)
		}
	}
	return nil
}

func (m *mockTimelineRepoForHandler) CreateClip(ctx context.Context, c *domain.TimelineClip) error {
	m.clips[c.ID] = c
	c.ComputeTimings()
	return nil
}

func (m *mockTimelineRepoForHandler) GetClipByID(ctx context.Context, id string) (*domain.TimelineClip, error) {
	c, ok := m.clips[id]
	if !ok {
		return nil, domain.ErrClipNotFound
	}
	c.ComputeTimings()
	return c, nil
}

func (m *mockTimelineRepoForHandler) ListClipsByTrackID(ctx context.Context, trackID string) ([]*domain.TimelineClip, error) {
	var list []*domain.TimelineClip
	for _, c := range m.clips {
		if c.TrackID == trackID {
			c.ComputeTimings()
			list = append(list, c)
		}
	}
	return list, nil
}

func (m *mockTimelineRepoForHandler) ListClipsByTimelineID(ctx context.Context, timelineID string) ([]*domain.TimelineClip, error) {
	var list []*domain.TimelineClip
	for _, c := range m.clips {
		if tr, ok := m.tracks[c.TrackID]; ok && tr.TimelineID == timelineID {
			c.ComputeTimings()
			list = append(list, c)
		}
	}
	return list, nil
}

func (m *mockTimelineRepoForHandler) UpdateClip(ctx context.Context, c *domain.TimelineClip) error {
	if _, ok := m.clips[c.ID]; !ok {
		return domain.ErrClipNotFound
	}
	c.ComputeTimings()
	m.clips[c.ID] = c
	return nil
}

func (m *mockTimelineRepoForHandler) DeleteClip(ctx context.Context, id string) error {
	if _, ok := m.clips[id]; !ok {
		return domain.ErrClipNotFound
	}
	delete(m.clips, id)
	return nil
}

func (m *mockTimelineRepoForHandler) GetFullTimeline(ctx context.Context, projectID string) (*domain.Timeline, error) {
	tl, err := m.GetTimelineByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	tracks, _ := m.ListTracksByTimelineID(ctx, tl.ID)
	for _, tr := range tracks {
		clips, _ := m.ListClipsByTrackID(ctx, tr.ID)
		tr.Clips = clips
	}
	tl.Tracks = tracks
	return tl, nil
}

func setupTestTimelineHandler() (http.Handler, *mockTimelineRepoForHandler, *inMemoryRepo, *mockMediaRepoForHandler) {
	tlRepo := newMockTimelineRepoForHandler()
	projectRepo := newInMemoryRepo()
	mediaRepo := newMockMediaRepoForHandler()

	_ = projectRepo.Create(context.Background(), &domain.Project{
		ID:   "proj-tl",
		Name: "Timeline Project",
	})

	_ = mediaRepo.Create(context.Background(), &domain.MediaAsset{
		ID:        "media-tl-1",
		ProjectID: "proj-tl",
		MimeType:  "video/mp4",
		Duration:  60.0,
	})

	svc := service.NewTimelineService(tlRepo, projectRepo, mediaRepo)
	h := NewTimelineHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("/projects/", func(w http.ResponseWriter, r *http.Request) {
		h.TimelineDispatcher(w, r)
	})

	return mux, tlRepo, projectRepo, mediaRepo
}

func TestTimelineHandler_FullHTTPFlow(t *testing.T) {
	mux, _, _, _ := setupTestTimelineHandler()

	var trackID string
	var clipID string

	t.Run("GET /projects/:id/timeline auto-creates default timeline 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/proj-tl/timeline", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		var tl domain.Timeline
		if err := json.Unmarshal(rr.Body.Bytes(), &tl); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if tl.Name != "Main Timeline" || len(tl.Tracks) != 2 {
			t.Errorf("unexpected timeline response: %+v", tl)
		}
		trackID = tl.Tracks[0].ID
	})

	t.Run("POST /projects/:id/timeline/tracks creates track 201", func(t *testing.T) {
		body := `{"type":"VIDEO","name":"Video 2","trackOrder":2}`
		req := httptest.NewRequest(http.MethodPost, "/projects/proj-tl/timeline/tracks", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var tr domain.Track
		if err := json.Unmarshal(rr.Body.Bytes(), &tr); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if tr.Name != "Video 2" || tr.Type != domain.TrackTypeVideo {
			t.Errorf("unexpected track created: %+v", tr)
		}
	})

	t.Run("PATCH /projects/:id/timeline/tracks/:trackId updates track 200", func(t *testing.T) {
		body := `{"name":"A-Roll Video"}`
		req := httptest.NewRequest(http.MethodPatch, "/projects/proj-tl/timeline/tracks/"+trackID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		var tr domain.Track
		if err := json.Unmarshal(rr.Body.Bytes(), &tr); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if tr.Name != "A-Roll Video" {
			t.Errorf("expected name 'A-Roll Video', got '%s'", tr.Name)
		}
	})

	t.Run("POST /projects/:id/timeline/tracks/:trackId/clips creates clip 201", func(t *testing.T) {
		body := `{"mediaAssetId":"media-tl-1","timelineStart":0,"sourceIn":0,"sourceOut":5000000}`
		req := httptest.NewRequest(http.MethodPost, "/projects/proj-tl/timeline/tracks/"+trackID+"/clips", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rr.Code, rr.Body.String())
		}

		var c domain.TimelineClip
		if err := json.Unmarshal(rr.Body.Bytes(), &c); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if c.Duration != 5000000 || c.TimelineEnd != 5000000 {
			t.Errorf("unexpected clip timings: %+v", c)
		}
		clipID = c.ID
	})

	t.Run("POST /projects/:id/timeline/tracks/:trackId/clips rejects overlap 409", func(t *testing.T) {
		// Overlaps with [0, 5s)
		body := `{"mediaAssetId":"media-tl-1","timelineStart":2000000,"sourceIn":0,"sourceOut":4000000}`
		req := httptest.NewRequest(http.MethodPost, "/projects/proj-tl/timeline/tracks/"+trackID+"/clips", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict for overlapping clip, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("PATCH /projects/:id/timeline/tracks/:trackId/clips/:clipId updates clip 200", func(t *testing.T) {
		body := `{"timelineStart":10000000}`
		req := httptest.NewRequest(http.MethodPatch, "/projects/proj-tl/timeline/tracks/"+trackID+"/clips/"+clipID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", rr.Code, rr.Body.String())
		}

		var c domain.TimelineClip
		if err := json.Unmarshal(rr.Body.Bytes(), &c); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if c.TimelineStart != 10000000 {
			t.Errorf("expected timelineStart 10000000, got %d", c.TimelineStart)
		}
	})

	t.Run("DELETE /projects/:id/timeline/tracks/:trackId/clips/:clipId deletes clip 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/projects/proj-tl/timeline/tracks/"+trackID+"/clips/"+clipID, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("DELETE /projects/:id/timeline/tracks/:trackId deletes track 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/projects/proj-tl/timeline/tracks/"+trackID, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Fatalf("expected 204 No Content, got %d: %s", rr.Code, rr.Body.String())
		}
	})

	t.Run("GET /projects/:missing/timeline returns 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/proj-missing/timeline", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", rr.Code)
		}
	})
}
