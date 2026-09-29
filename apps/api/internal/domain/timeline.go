package domain

import (
	"errors"
	"strings"
	"time"
)

type TrackType string

const (
	TrackTypeVideo TrackType = "VIDEO"
	TrackTypeAudio TrackType = "AUDIO"
)

func (t TrackType) IsValid() bool {
	return t == TrackTypeVideo || t == TrackTypeAudio
}

var (
	ErrTimelineNotFound           = errors.New("timeline not found")
	ErrTrackNotFound              = errors.New("track not found")
	ErrClipNotFound               = errors.New("timeline clip not found")
	ErrInvalidTimelineID          = errors.New("invalid timeline id")
	ErrInvalidTrackID             = errors.New("invalid track id")
	ErrInvalidClipID              = errors.New("invalid clip id")
	ErrInvalidTrackType           = errors.New("invalid track type: must be VIDEO or AUDIO")
	ErrInvalidTimelinePosition    = errors.New("timeline_start must be greater than or equal to 0")
	ErrInvalidSourceRange         = errors.New("source_out must be strictly greater than source_in and source_in must be non-negative")
	ErrSourceOutExceedsMedia      = errors.New("source_out exceeds media asset duration")
	ErrIncompatibleMediaTrack     = errors.New("media asset is incompatible with the destination track type")
	ErrClipOverlap                = errors.New("clips on the same track must not overlap")
	ErrTimelineAlreadyExists      = errors.New("timeline already exists for this project")
	ErrTimelineNotBelongToProject = errors.New("timeline does not belong to the specified project")
	ErrTrackNotBelongToTimeline   = errors.New("track does not belong to the specified timeline")
	ErrClipNotBelongToTrack       = errors.New("clip does not belong to the specified track")
)

// Timebase conversion constants (Microseconds)
const MicrosecondsPerSecond int64 = 1_000_000

func SecondsToMicroseconds(seconds float64) int64 {
	return int64(seconds * float64(MicrosecondsPerSecond))
}

func MicrosecondsToSeconds(micros int64) float64 {
	return float64(micros) / float64(MicrosecondsPerSecond)
}

type Timeline struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Tracks    []*Track  `json:"tracks,omitempty"`
}

type Track struct {
	ID         string          `json:"id"`
	TimelineID string          `json:"timelineId"`
	Type       TrackType       `json:"type"`
	Name       string          `json:"name"`
	TrackOrder int             `json:"trackOrder"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
	Clips      []*TimelineClip `json:"clips,omitempty"`
}

type TimelineClip struct {
	ID            string      `json:"id"`
	TrackID       string      `json:"trackId"`
	MediaAssetID  string      `json:"mediaAssetId"`
	TimelineStart int64       `json:"timelineStart"` // Microseconds
	SourceIn      int64       `json:"sourceIn"`      // Microseconds
	SourceOut     int64       `json:"sourceOut"`     // Microseconds
	Duration      int64       `json:"duration"`      // Computed duration (sourceOut - sourceIn) in microseconds
	TimelineEnd   int64       `json:"timelineEnd"`   // Computed end (timelineStart + duration) in microseconds
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
	Media         *MediaAsset `json:"media,omitempty"`
}

func (c *TimelineClip) ComputeTimings() {
	c.Duration = c.SourceOut - c.SourceIn
	c.TimelineEnd = c.TimelineStart + c.Duration
}

type CreateTimelineInput struct {
	Name string `json:"name"`
}

func (in *CreateTimelineInput) Validate() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		in.Name = "Main Timeline"
	}
	if len(in.Name) > 255 {
		return errors.New("timeline name cannot exceed 255 characters")
	}
	return nil
}

type CreateTrackInput struct {
	Type       TrackType `json:"type"`
	Name       string    `json:"name"`
	TrackOrder int       `json:"trackOrder"`
}

func (in *CreateTrackInput) Validate() error {
	if !in.Type.IsValid() {
		return ErrInvalidTrackType
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		if in.Type == TrackTypeVideo {
			in.Name = "Video Track"
		} else {
			in.Name = "Audio Track"
		}
	}
	if len(in.Name) > 255 {
		return errors.New("track name cannot exceed 255 characters")
	}
	return nil
}

type UpdateTrackInput struct {
	Name       *string `json:"name,omitempty"`
	TrackOrder *int    `json:"trackOrder,omitempty"`
}

func (in *UpdateTrackInput) Validate() error {
	if in.Name != nil {
		trimmed := strings.TrimSpace(*in.Name)
		if trimmed == "" {
			return errors.New("track name cannot be empty")
		}
		if len(trimmed) > 255 {
			return errors.New("track name cannot exceed 255 characters")
		}
		*in.Name = trimmed
	}
	return nil
}

type CreateClipInput struct {
	MediaAssetID  string `json:"mediaAssetId"`
	TimelineStart int64  `json:"timelineStart"`
	SourceIn      int64  `json:"sourceIn"`
	SourceOut     int64  `json:"sourceOut"`
}

func (in *CreateClipInput) Validate() error {
	in.MediaAssetID = strings.TrimSpace(in.MediaAssetID)
	if in.MediaAssetID == "" {
		return errors.New("mediaAssetId is required")
	}
	if in.TimelineStart < 0 {
		return ErrInvalidTimelinePosition
	}
	if in.SourceIn < 0 || in.SourceOut <= in.SourceIn {
		return ErrInvalidSourceRange
	}
	return nil
}

type UpdateClipInput struct {
	TimelineStart *int64 `json:"timelineStart,omitempty"`
	SourceIn      *int64 `json:"sourceIn,omitempty"`
	SourceOut     *int64 `json:"sourceOut,omitempty"`
}

func (in *UpdateClipInput) Validate() error {
	if in.TimelineStart != nil && *in.TimelineStart < 0 {
		return ErrInvalidTimelinePosition
	}
	if in.SourceIn != nil && *in.SourceIn < 0 {
		return ErrInvalidSourceRange
	}
	return nil
}

// ClipsOverlap checks if two clip intervals on the same track intersect in timeline time:
// Interval A: [startA, startA + durA)
// Interval B: [startB, startB + durB)
func ClipsOverlap(startA, durA, startB, durB int64) bool {
	if durA <= 0 || durB <= 0 {
		return false
	}
	endA := startA + durA
	endB := startB + durB

	maxStart := startA
	if startB > maxStart {
		maxStart = startB
	}

	minEnd := endA
	if endB < minEnd {
		minEnd = endB
	}

	return maxStart < minEnd
}

// ValidateClipInvariants checks time bounds, media duration compatibility, and track type compatibility.
func ValidateClipInvariants(
	track *Track,
	media *MediaAsset,
	timelineStart, sourceIn, sourceOut int64,
) error {
	if timelineStart < 0 {
		return ErrInvalidTimelinePosition
	}
	if sourceIn < 0 || sourceOut <= sourceIn {
		return ErrInvalidSourceRange
	}

	// Media duration check if metadata duration is known (> 0)
	if media != nil && media.Duration > 0 {
		mediaDurationMicros := SecondsToMicroseconds(media.Duration)
		// Allow a small epsilon (e.g. 50ms = 50,000 micros) for rounding differences in container headers
		if sourceOut > mediaDurationMicros+50_000 {
			return ErrSourceOutExceedsMedia
		}
	}

	// Track type compatibility check
	if track != nil && media != nil {
		if track.Type == TrackTypeVideo {
			// Video track requires video MIME type
			if !strings.HasPrefix(strings.ToLower(media.MimeType), "video/") && media.MimeType != "application/octet-stream" {
				return ErrIncompatibleMediaTrack
			}
		}
		// Audio tracks can accept video or audio container formats
	}

	return nil
}
