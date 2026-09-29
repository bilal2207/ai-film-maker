package repository

import (
	"context"

	"github.com/ai-filmmaker/api/internal/domain"
)

type TimelineRepository interface {
	// Timeline
	CreateTimeline(ctx context.Context, timeline *domain.Timeline) error
	GetTimelineByProjectID(ctx context.Context, projectID string) (*domain.Timeline, error)
	GetTimelineByID(ctx context.Context, id string) (*domain.Timeline, error)
	UpdateTimeline(ctx context.Context, timeline *domain.Timeline) error
	DeleteTimeline(ctx context.Context, id string) error

	// Tracks
	CreateTrack(ctx context.Context, track *domain.Track) error
	GetTrackByID(ctx context.Context, id string) (*domain.Track, error)
	ListTracksByTimelineID(ctx context.Context, timelineID string) ([]*domain.Track, error)
	UpdateTrack(ctx context.Context, track *domain.Track) error
	DeleteTrack(ctx context.Context, id string) error

	// Clips
	CreateClip(ctx context.Context, clip *domain.TimelineClip) error
	GetClipByID(ctx context.Context, id string) (*domain.TimelineClip, error)
	ListClipsByTrackID(ctx context.Context, trackID string) ([]*domain.TimelineClip, error)
	ListClipsByTimelineID(ctx context.Context, timelineID string) ([]*domain.TimelineClip, error)
	UpdateClip(ctx context.Context, clip *domain.TimelineClip) error
	DeleteClip(ctx context.Context, id string) error

	// Full Hierarchy query
	GetFullTimeline(ctx context.Context, projectID string) (*domain.Timeline, error)
}
