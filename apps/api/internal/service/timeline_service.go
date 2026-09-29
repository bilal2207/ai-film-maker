package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ai-filmmaker/api/internal/domain"
	"github.com/ai-filmmaker/api/internal/repository"
	"github.com/google/uuid"
)

type TimelineService struct {
	timelineRepo repository.TimelineRepository
	projectRepo  repository.ProjectRepository
	mediaRepo    repository.MediaRepository
}

func NewTimelineService(
	timelineRepo repository.TimelineRepository,
	projectRepo repository.ProjectRepository,
	mediaRepo repository.MediaRepository,
) *TimelineService {
	return &TimelineService{
		timelineRepo: timelineRepo,
		projectRepo:  projectRepo,
		mediaRepo:    mediaRepo,
	}
}

// GetOrCreateTimeline retrieves the project's timeline or initializes it with default tracks if none exists.
func (s *TimelineService) GetOrCreateTimeline(ctx context.Context, projectID string, input *domain.CreateTimelineInput) (*domain.Timeline, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}

	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	tl, err := s.timelineRepo.GetFullTimeline(ctx, projectID)
	if err == nil {
		return tl, nil
	}

	if !errors.Is(err, domain.ErrTimelineNotFound) {
		return nil, err
	}

	name := "Main Timeline"
	if input != nil && strings.TrimSpace(input.Name) != "" {
		name = strings.TrimSpace(input.Name)
	}

	now := time.Now().UTC()
	newTL := &domain.Timeline{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.timelineRepo.CreateTimeline(ctx, newTL); err != nil {
		return nil, fmt.Errorf("failed to create primary timeline: %w", err)
	}

	// Create initial default tracks: Video 1 and Audio 1
	videoTrack := &domain.Track{
		ID:         uuid.New().String(),
		TimelineID: newTL.ID,
		Type:       domain.TrackTypeVideo,
		Name:       "Video 1",
		TrackOrder: 0,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_ = s.timelineRepo.CreateTrack(ctx, videoTrack)

	audioTrack := &domain.Track{
		ID:         uuid.New().String(),
		TimelineID: newTL.ID,
		Type:       domain.TrackTypeAudio,
		Name:       "Audio 1",
		TrackOrder: 1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_ = s.timelineRepo.CreateTrack(ctx, audioTrack)

	return s.timelineRepo.GetFullTimeline(ctx, projectID)
}

func (s *TimelineService) CreateTimeline(ctx context.Context, projectID string, input domain.CreateTimelineInput) (*domain.Timeline, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}

	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check if already exists (one timeline per project)
	existing, err := s.timelineRepo.GetTimelineByProjectID(ctx, projectID)
	if err == nil && existing != nil {
		return nil, domain.ErrTimelineAlreadyExists
	}
	if err != nil && !errors.Is(err, domain.ErrTimelineNotFound) {
		return nil, err
	}

	now := time.Now().UTC()
	tl := &domain.Timeline{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Name:      input.Name,
		CreatedAt: now,
		UpdatedAt: now,
		Tracks:    []*domain.Track{},
	}

	if err := s.timelineRepo.CreateTimeline(ctx, tl); err != nil {
		return nil, fmt.Errorf("failed to create timeline: %w", err)
	}

	return tl, nil
}

func (s *TimelineService) GetTimeline(ctx context.Context, projectID string) (*domain.Timeline, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}

	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	return s.timelineRepo.GetFullTimeline(ctx, projectID)
}

// --- Tracks ---

func (s *TimelineService) CreateTrack(ctx context.Context, projectID string, input domain.CreateTrackInput) (*domain.Track, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}

	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	tl, err := s.timelineRepo.GetTimelineByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	track := &domain.Track{
		ID:         uuid.New().String(),
		TimelineID: tl.ID,
		Type:       input.Type,
		Name:       input.Name,
		TrackOrder: input.TrackOrder,
		CreatedAt:  now,
		UpdatedAt:  now,
		Clips:      []*domain.TimelineClip{},
	}

	if err := s.timelineRepo.CreateTrack(ctx, track); err != nil {
		return nil, fmt.Errorf("failed to create track: %w", err)
	}

	return track, nil
}

func (s *TimelineService) UpdateTrack(ctx context.Context, projectID, trackID string, input domain.UpdateTrackInput) (*domain.Track, error) {
	projectID = strings.TrimSpace(projectID)
	trackID = strings.TrimSpace(trackID)

	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}
	if trackID == "" {
		return nil, domain.ErrInvalidTrackID
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	tl, err := s.timelineRepo.GetTimelineByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	track, err := s.timelineRepo.GetTrackByID(ctx, trackID)
	if err != nil {
		return nil, err
	}

	if track.TimelineID != tl.ID {
		return nil, domain.ErrTrackNotBelongToTimeline
	}

	if input.Name != nil {
		track.Name = *input.Name
	}
	if input.TrackOrder != nil {
		track.TrackOrder = *input.TrackOrder
	}
	track.UpdatedAt = time.Now().UTC()

	if err := s.timelineRepo.UpdateTrack(ctx, track); err != nil {
		return nil, fmt.Errorf("failed to update track: %w", err)
	}

	return track, nil
}

func (s *TimelineService) DeleteTrack(ctx context.Context, projectID, trackID string) error {
	projectID = strings.TrimSpace(projectID)
	trackID = strings.TrimSpace(trackID)

	if projectID == "" {
		return domain.ErrInvalidProjectID
	}
	if trackID == "" {
		return domain.ErrInvalidTrackID
	}

	tl, err := s.timelineRepo.GetTimelineByProjectID(ctx, projectID)
	if err != nil {
		return err
	}

	track, err := s.timelineRepo.GetTrackByID(ctx, trackID)
	if err != nil {
		return err
	}

	if track.TimelineID != tl.ID {
		return domain.ErrTrackNotBelongToTimeline
	}

	return s.timelineRepo.DeleteTrack(ctx, trackID)
}

// --- Clips ---

func (s *TimelineService) CreateClip(ctx context.Context, projectID, trackID string, input domain.CreateClipInput) (*domain.TimelineClip, error) {
	projectID = strings.TrimSpace(projectID)
	trackID = strings.TrimSpace(trackID)

	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}
	if trackID == "" {
		return nil, domain.ErrInvalidTrackID
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// 1. Verify timeline ownership
	tl, err := s.timelineRepo.GetTimelineByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 2. Verify track belongs to timeline
	track, err := s.timelineRepo.GetTrackByID(ctx, trackID)
	if err != nil {
		return nil, err
	}
	if track.TimelineID != tl.ID {
		return nil, domain.ErrTrackNotBelongToTimeline
	}

	// 3. Verify media asset exists and belongs to the same project
	media, err := s.mediaRepo.GetByID(ctx, input.MediaAssetID)
	if err != nil {
		return nil, err
	}
	if media.ProjectID != projectID {
		return nil, domain.ErrMediaNotBelongToProject
	}

	// 4. Validate time and track compatibility invariants
	if err := domain.ValidateClipInvariants(track, media, input.TimelineStart, input.SourceIn, input.SourceOut); err != nil {
		return nil, err
	}

	duration := input.SourceOut - input.SourceIn

	// 5. Enforce same-track overlap policy
	existingClips, err := s.timelineRepo.ListClipsByTrackID(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to list track clips for overlap verification: %w", err)
	}

	for _, c := range existingClips {
		if domain.ClipsOverlap(c.TimelineStart, c.Duration, input.TimelineStart, duration) {
			return nil, domain.ErrClipOverlap
		}
	}

	// 6. Create clip
	now := time.Now().UTC()
	clip := &domain.TimelineClip{
		ID:            uuid.New().String(),
		TrackID:       trackID,
		MediaAssetID:  input.MediaAssetID,
		TimelineStart: input.TimelineStart,
		SourceIn:      input.SourceIn,
		SourceOut:     input.SourceOut,
		CreatedAt:     now,
		UpdatedAt:     now,
		Media:         media,
	}
	clip.ComputeTimings()

	if err := s.timelineRepo.CreateClip(ctx, clip); err != nil {
		return nil, fmt.Errorf("failed to create timeline clip: %w", err)
	}

	return clip, nil
}

func (s *TimelineService) UpdateClip(ctx context.Context, projectID, trackID, clipID string, input domain.UpdateClipInput) (*domain.TimelineClip, error) {
	projectID = strings.TrimSpace(projectID)
	trackID = strings.TrimSpace(trackID)
	clipID = strings.TrimSpace(clipID)

	if projectID == "" {
		return nil, domain.ErrInvalidProjectID
	}
	if trackID == "" {
		return nil, domain.ErrInvalidTrackID
	}
	if clipID == "" {
		return nil, domain.ErrInvalidClipID
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// 1. Verify timeline ownership
	tl, err := s.timelineRepo.GetTimelineByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// 2. Verify track belongs to timeline
	track, err := s.timelineRepo.GetTrackByID(ctx, trackID)
	if err != nil {
		return nil, err
	}
	if track.TimelineID != tl.ID {
		return nil, domain.ErrTrackNotBelongToTimeline
	}

	// 3. Fetch clip and verify track ownership
	clip, err := s.timelineRepo.GetClipByID(ctx, clipID)
	if err != nil {
		return nil, err
	}
	if clip.TrackID != track.ID {
		return nil, domain.ErrClipNotBelongToTrack
	}

	// 4. Apply updates
	newStart := clip.TimelineStart
	newSourceIn := clip.SourceIn
	newSourceOut := clip.SourceOut

	if input.TimelineStart != nil {
		newStart = *input.TimelineStart
	}
	if input.SourceIn != nil {
		newSourceIn = *input.SourceIn
	}
	if input.SourceOut != nil {
		newSourceOut = *input.SourceOut
	}

	// 5. Validate invariants with updated values
	if err := domain.ValidateClipInvariants(track, clip.Media, newStart, newSourceIn, newSourceOut); err != nil {
		return nil, err
	}

	newDuration := newSourceOut - newSourceIn

	// 6. Check overlap against other clips on this track
	existingClips, err := s.timelineRepo.ListClipsByTrackID(ctx, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to list track clips: %w", err)
	}

	for _, c := range existingClips {
		if c.ID == clip.ID {
			continue // skip self
		}
		if domain.ClipsOverlap(c.TimelineStart, c.Duration, newStart, newDuration) {
			return nil, domain.ErrClipOverlap
		}
	}

	clip.TimelineStart = newStart
	clip.SourceIn = newSourceIn
	clip.SourceOut = newSourceOut
	clip.UpdatedAt = time.Now().UTC()
	clip.ComputeTimings()

	if err := s.timelineRepo.UpdateClip(ctx, clip); err != nil {
		return nil, fmt.Errorf("failed to update timeline clip: %w", err)
	}

	return clip, nil
}

func (s *TimelineService) DeleteClip(ctx context.Context, projectID, trackID, clipID string) error {
	projectID = strings.TrimSpace(projectID)
	trackID = strings.TrimSpace(trackID)
	clipID = strings.TrimSpace(clipID)

	if projectID == "" {
		return domain.ErrInvalidProjectID
	}
	if trackID == "" {
		return domain.ErrInvalidTrackID
	}
	if clipID == "" {
		return domain.ErrInvalidClipID
	}

	tl, err := s.timelineRepo.GetTimelineByProjectID(ctx, projectID)
	if err != nil {
		return err
	}

	track, err := s.timelineRepo.GetTrackByID(ctx, trackID)
	if err != nil {
		return err
	}
	if track.TimelineID != tl.ID {
		return domain.ErrTrackNotBelongToTimeline
	}

	clip, err := s.timelineRepo.GetClipByID(ctx, clipID)
	if err != nil {
		return err
	}
	if clip.TrackID != track.ID {
		return domain.ErrClipNotBelongToTrack
	}

	return s.timelineRepo.DeleteClip(ctx, clipID)
}
