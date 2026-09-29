package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ai-filmmaker/api/internal/domain"
)

type mockTimelineRepo struct {
	timelines map[string]*domain.Timeline
	tracks    map[string]*domain.Track
	clips     map[string]*domain.TimelineClip
}

func newMockTimelineRepo() *mockTimelineRepo {
	return &mockTimelineRepo{
		timelines: make(map[string]*domain.Timeline),
		tracks:    make(map[string]*domain.Track),
		clips:     make(map[string]*domain.TimelineClip),
	}
}

func (m *mockTimelineRepo) CreateTimeline(ctx context.Context, t *domain.Timeline) error {
	m.timelines[t.ID] = t
	return nil
}

func (m *mockTimelineRepo) GetTimelineByProjectID(ctx context.Context, projectID string) (*domain.Timeline, error) {
	for _, t := range m.timelines {
		if t.ProjectID == projectID {
			return t, nil
		}
	}
	return nil, domain.ErrTimelineNotFound
}

func (m *mockTimelineRepo) GetTimelineByID(ctx context.Context, id string) (*domain.Timeline, error) {
	t, ok := m.timelines[id]
	if !ok {
		return nil, domain.ErrTimelineNotFound
	}
	return t, nil
}

func (m *mockTimelineRepo) UpdateTimeline(ctx context.Context, t *domain.Timeline) error {
	if _, ok := m.timelines[t.ID]; !ok {
		return domain.ErrTimelineNotFound
	}
	m.timelines[t.ID] = t
	return nil
}

func (m *mockTimelineRepo) DeleteTimeline(ctx context.Context, id string) error {
	if _, ok := m.timelines[id]; !ok {
		return domain.ErrTimelineNotFound
	}
	delete(m.timelines, id)
	return nil
}

func (m *mockTimelineRepo) CreateTrack(ctx context.Context, tr *domain.Track) error {
	m.tracks[tr.ID] = tr
	return nil
}

func (m *mockTimelineRepo) GetTrackByID(ctx context.Context, id string) (*domain.Track, error) {
	tr, ok := m.tracks[id]
	if !ok {
		return nil, domain.ErrTrackNotFound
	}
	return tr, nil
}

func (m *mockTimelineRepo) ListTracksByTimelineID(ctx context.Context, timelineID string) ([]*domain.Track, error) {
	var list []*domain.Track
	for _, tr := range m.tracks {
		if tr.TimelineID == timelineID {
			list = append(list, tr)
		}
	}
	return list, nil
}

func (m *mockTimelineRepo) UpdateTrack(ctx context.Context, tr *domain.Track) error {
	if _, ok := m.tracks[tr.ID]; !ok {
		return domain.ErrTrackNotFound
	}
	m.tracks[tr.ID] = tr
	return nil
}

func (m *mockTimelineRepo) DeleteTrack(ctx context.Context, id string) error {
	if _, ok := m.tracks[id]; !ok {
		return domain.ErrTrackNotFound
	}
	delete(m.tracks, id)
	// Cascading delete clips on this track
	for cid, c := range m.clips {
		if c.TrackID == id {
			delete(m.clips, cid)
		}
	}
	return nil
}

func (m *mockTimelineRepo) CreateClip(ctx context.Context, c *domain.TimelineClip) error {
	m.clips[c.ID] = c
	c.ComputeTimings()
	return nil
}

func (m *mockTimelineRepo) GetClipByID(ctx context.Context, id string) (*domain.TimelineClip, error) {
	c, ok := m.clips[id]
	if !ok {
		return nil, domain.ErrClipNotFound
	}
	c.ComputeTimings()
	return c, nil
}

func (m *mockTimelineRepo) ListClipsByTrackID(ctx context.Context, trackID string) ([]*domain.TimelineClip, error) {
	var list []*domain.TimelineClip
	for _, c := range m.clips {
		if c.TrackID == trackID {
			c.ComputeTimings()
			list = append(list, c)
		}
	}
	return list, nil
}

func (m *mockTimelineRepo) ListClipsByTimelineID(ctx context.Context, timelineID string) ([]*domain.TimelineClip, error) {
	var list []*domain.TimelineClip
	for _, c := range m.clips {
		if tr, ok := m.tracks[c.TrackID]; ok && tr.TimelineID == timelineID {
			c.ComputeTimings()
			list = append(list, c)
		}
	}
	return list, nil
}

func (m *mockTimelineRepo) UpdateClip(ctx context.Context, c *domain.TimelineClip) error {
	if _, ok := m.clips[c.ID]; !ok {
		return domain.ErrClipNotFound
	}
	c.ComputeTimings()
	m.clips[c.ID] = c
	return nil
}

func (m *mockTimelineRepo) DeleteClip(ctx context.Context, id string) error {
	if _, ok := m.clips[id]; !ok {
		return domain.ErrClipNotFound
	}
	delete(m.clips, id)
	return nil
}

func (m *mockTimelineRepo) GetFullTimeline(ctx context.Context, projectID string) (*domain.Timeline, error) {
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

func setupTestTimelineService() (*TimelineService, *mockTimelineRepo, *mockProjectRepo, *mockMediaRepo) {
	tlRepo := newMockTimelineRepo()
	pRepo := newMockProjectRepo()
	mRepo := newMockMediaRepo()

	svc := NewTimelineService(tlRepo, pRepo, mRepo)
	return svc, tlRepo, pRepo, mRepo
}

func TestTimelineService_GetOrCreateTimeline(t *testing.T) {
	svc, _, pRepo, _ := setupTestTimelineService()
	ctx := context.Background()

	_ = pRepo.Create(ctx, &domain.Project{ID: "proj-10", Name: "Epic Film"})

	t.Run("auto-creates timeline with default tracks when missing", func(t *testing.T) {
		tl, err := svc.GetOrCreateTimeline(ctx, "proj-10", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tl.Name != "Main Timeline" {
			t.Errorf("expected name 'Main Timeline', got '%s'", tl.Name)
		}
		if len(tl.Tracks) != 2 {
			t.Errorf("expected 2 default tracks (Video 1 and Audio 1), got %d", len(tl.Tracks))
		}
	})

	t.Run("subsequent call retrieves existing timeline", func(t *testing.T) {
		tl, err := svc.GetOrCreateTimeline(ctx, "proj-10", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if tl.ProjectID != "proj-10" {
			t.Errorf("expected project_id 'proj-10', got '%s'", tl.ProjectID)
		}
	})
}

func TestTimelineService_TrackOperations(t *testing.T) {
	svc, _, pRepo, _ := setupTestTimelineService()
	ctx := context.Background()

	_ = pRepo.Create(ctx, &domain.Project{ID: "proj-20", Name: "Action Movie"})
	tl, _ := svc.GetOrCreateTimeline(ctx, "proj-20", nil)

	t.Run("create custom track", func(t *testing.T) {
		track, err := svc.CreateTrack(ctx, "proj-20", domain.CreateTrackInput{
			Type:       domain.TrackTypeVideo,
			Name:       "VFX Overlay",
			TrackOrder: 2,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if track.Name != "VFX Overlay" || track.TimelineID != tl.ID {
			t.Errorf("unexpected track created: %+v", track)
		}
	})

	t.Run("create track with invalid type fails", func(t *testing.T) {
		_, err := svc.CreateTrack(ctx, "proj-20", domain.CreateTrackInput{
			Type: "SUBTITLE",
			Name: "Subtitles",
		})
		if !errors.Is(err, domain.ErrInvalidTrackType) {
			t.Errorf("expected ErrInvalidTrackType, got %v", err)
		}
	})

	t.Run("update track name and order", func(t *testing.T) {
		track, _ := svc.CreateTrack(ctx, "proj-20", domain.CreateTrackInput{
			Type:       domain.TrackTypeAudio,
			Name:       "SFX",
			TrackOrder: 3,
		})

		newName := "SFX & Foley"
		newOrder := 5
		updated, err := svc.UpdateTrack(ctx, "proj-20", track.ID, domain.UpdateTrackInput{
			Name:       &newName,
			TrackOrder: &newOrder,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != "SFX & Foley" || updated.TrackOrder != 5 {
			t.Errorf("unexpected track update: %+v", updated)
		}
	})

	t.Run("delete track", func(t *testing.T) {
		track, _ := svc.CreateTrack(ctx, "proj-20", domain.CreateTrackInput{
			Type: domain.TrackTypeAudio,
			Name: "Temp Track",
		})

		err := svc.DeleteTrack(ctx, "proj-20", track.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify deletion
		err = svc.DeleteTrack(ctx, "proj-20", track.ID)
		if !errors.Is(err, domain.ErrTrackNotFound) {
			t.Errorf("expected ErrTrackNotFound on repeated delete, got %v", err)
		}
	})
}

func TestTimelineService_ClipOperationsAndOverlap(t *testing.T) {
	svc, _, pRepo, mRepo := setupTestTimelineService()
	ctx := context.Background()

	_ = pRepo.Create(ctx, &domain.Project{ID: "p-clip", Name: "Clip Test Project"})
	_ = pRepo.Create(ctx, &domain.Project{ID: "p-other", Name: "Other Project"})

	tl, _ := svc.GetOrCreateTimeline(ctx, "p-clip", nil)
	videoTrack1 := tl.Tracks[0] // TrackTypeVideo
	audioTrack1 := tl.Tracks[1] // TrackTypeAudio

	// Create media assets
	videoMedia := &domain.MediaAsset{
		ID:        "m-vid-1",
		ProjectID: "p-clip",
		MimeType:  "video/mp4",
		Duration:  30.0, // 30,000,000 micros
	}
	_ = mRepo.Create(ctx, videoMedia)

	otherProjectMedia := &domain.MediaAsset{
		ID:        "m-other-1",
		ProjectID: "p-other",
		MimeType:  "video/mp4",
		Duration:  30.0,
	}
	_ = mRepo.Create(ctx, otherProjectMedia)

	t.Run("create clip success", func(t *testing.T) {
		// Clip A: [0, 5_000_000)
		clip, err := svc.CreateClip(ctx, "p-clip", videoTrack1.ID, domain.CreateClipInput{
			MediaAssetID:  videoMedia.ID,
			TimelineStart: 0,
			SourceIn:      0,
			SourceOut:     5_000_000, // 5s
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if clip.Duration != 5_000_000 || clip.TimelineEnd != 5_000_000 {
			t.Errorf("unexpected timings: duration=%d, end=%d", clip.Duration, clip.TimelineEnd)
		}
	})

	t.Run("cross-project media reference rejected", func(t *testing.T) {
		_, err := svc.CreateClip(ctx, "p-clip", videoTrack1.ID, domain.CreateClipInput{
			MediaAssetID:  otherProjectMedia.ID,
			TimelineStart: 6_000_000,
			SourceIn:      0,
			SourceOut:     5_000_000,
		})
		if !errors.Is(err, domain.ErrMediaNotBelongToProject) {
			t.Errorf("expected ErrMediaNotBelongToProject, got %v", err)
		}
	})

	t.Run("overlapping clip on same track rejected", func(t *testing.T) {
		// Clip A is at [0, 5_000_000)
		// Try placing Clip B at [2_000_000, 7_000_000) -> overlaps!
		_, err := svc.CreateClip(ctx, "p-clip", videoTrack1.ID, domain.CreateClipInput{
			MediaAssetID:  videoMedia.ID,
			TimelineStart: 2_000_000,
			SourceIn:      0,
			SourceOut:     5_000_000,
		})
		if !errors.Is(err, domain.ErrClipOverlap) {
			t.Errorf("expected ErrClipOverlap on same track, got %v", err)
		}
	})

	t.Run("abutting clip on same track allowed", func(t *testing.T) {
		// Clip A is at [0, 5_000_000)
		// Clip C starts exactly at 5_000_000: [5_000_000, 10_000_000)
		clip, err := svc.CreateClip(ctx, "p-clip", videoTrack1.ID, domain.CreateClipInput{
			MediaAssetID:  videoMedia.ID,
			TimelineStart: 5_000_000,
			SourceIn:      0,
			SourceOut:     5_000_000,
		})
		if err != nil {
			t.Fatalf("expected abutting clip to succeed, got %v", err)
		}
		if clip.TimelineStart != 5_000_000 {
			t.Errorf("expected start at 5000000, got %d", clip.TimelineStart)
		}
	})

	t.Run("overlapping clips on DIFFERENT tracks allowed", func(t *testing.T) {
		// Video track has clips at [0, 5s) and [5s, 10s)
		// Placing audio clip at [0, 10s) on Audio track should succeed!
		clip, err := svc.CreateClip(ctx, "p-clip", audioTrack1.ID, domain.CreateClipInput{
			MediaAssetID:  videoMedia.ID,
			TimelineStart: 0,
			SourceIn:      0,
			SourceOut:     10_000_000,
		})
		if err != nil {
			t.Fatalf("expected different track clip to succeed, got %v", err)
		}
		if clip.TrackID != audioTrack1.ID {
			t.Errorf("expected clip on audio track")
		}
	})

	t.Run("update clip position and duration", func(t *testing.T) {
		// Create a separate track for update testing
		tr, _ := svc.CreateTrack(ctx, "p-clip", domain.CreateTrackInput{
			Type: domain.TrackTypeVideo,
			Name: "V2",
		})

		clip1, _ := svc.CreateClip(ctx, "p-clip", tr.ID, domain.CreateClipInput{
			MediaAssetID:  videoMedia.ID,
			TimelineStart: 0,
			SourceIn:      0,
			SourceOut:     3_000_000, // [0, 3s)
		})

		clip2, _ := svc.CreateClip(ctx, "p-clip", tr.ID, domain.CreateClipInput{
			MediaAssetID:  videoMedia.ID,
			TimelineStart: 10_000_000,
			SourceIn:      0,
			SourceOut:     3_000_000, // [10s, 13s)
		})

		// Moving clip2 to [2s, 5s) overlaps with clip1 [0, 3s) -> should fail!
		newStartBad := int64(2_000_000)
		_, err := svc.UpdateClip(ctx, "p-clip", tr.ID, clip2.ID, domain.UpdateClipInput{
			TimelineStart: &newStartBad,
		})
		if !errors.Is(err, domain.ErrClipOverlap) {
			t.Errorf("expected ErrClipOverlap on update, got %v", err)
		}

		// Moving clip2 to [4s, 7s) does not overlap -> should succeed!
		newStartGood := int64(4_000_000)
		updated, err := svc.UpdateClip(ctx, "p-clip", tr.ID, clip2.ID, domain.UpdateClipInput{
			TimelineStart: &newStartGood,
		})
		if err != nil {
			t.Fatalf("expected update to succeed, got %v", err)
		}
		if updated.TimelineStart != 4_000_000 {
			t.Errorf("expected timeline_start 4000000, got %d", updated.TimelineStart)
		}

		_ = clip1
	})

	t.Run("delete clip", func(t *testing.T) {
		tr, _ := svc.CreateTrack(ctx, "p-clip", domain.CreateTrackInput{
			Type: domain.TrackTypeVideo,
			Name: "V3",
		})
		clip, _ := svc.CreateClip(ctx, "p-clip", tr.ID, domain.CreateClipInput{
			MediaAssetID:  videoMedia.ID,
			TimelineStart: 0,
			SourceIn:      0,
			SourceOut:     2_000_000,
		})

		err := svc.DeleteClip(ctx, "p-clip", tr.ID, clip.ID)
		if err != nil {
			t.Fatalf("unexpected error deleting clip: %v", err)
		}

		// Second delete must fail
		err = svc.DeleteClip(ctx, "p-clip", tr.ID, clip.ID)
		if !errors.Is(err, domain.ErrClipNotFound) {
			t.Errorf("expected ErrClipNotFound, got %v", err)
		}
	})
}
