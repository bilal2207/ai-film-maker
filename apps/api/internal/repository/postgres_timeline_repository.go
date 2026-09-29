package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/ai-filmmaker/api/internal/domain"
)

type PostgresTimelineRepository struct {
	db *sql.DB
}

func NewPostgresTimelineRepository(db *sql.DB) *PostgresTimelineRepository {
	return &PostgresTimelineRepository{db: db}
}

// --- Timelines ---

func (r *PostgresTimelineRepository) CreateTimeline(ctx context.Context, t *domain.Timeline) error {
	query := `
		INSERT INTO timelines (id, project_id, name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5);
	`
	_, err := r.db.ExecContext(ctx, query, t.ID, t.ProjectID, t.Name, t.CreatedAt, t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert timeline: %w", err)
	}
	return nil
}

func (r *PostgresTimelineRepository) GetTimelineByProjectID(ctx context.Context, projectID string) (*domain.Timeline, error) {
	query := `
		SELECT id, project_id, name, created_at, updated_at
		FROM timelines
		WHERE project_id = $1;
	`
	var t domain.Timeline
	err := r.db.QueryRowContext(ctx, query, projectID).Scan(
		&t.ID, &t.ProjectID, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTimelineNotFound
		}
		return nil, fmt.Errorf("failed to get timeline by project id: %w", err)
	}
	return &t, nil
}

func (r *PostgresTimelineRepository) GetTimelineByID(ctx context.Context, id string) (*domain.Timeline, error) {
	query := `
		SELECT id, project_id, name, created_at, updated_at
		FROM timelines
		WHERE id = $1;
	`
	var t domain.Timeline
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.ProjectID, &t.Name, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTimelineNotFound
		}
		return nil, fmt.Errorf("failed to get timeline by id: %w", err)
	}
	return &t, nil
}

func (r *PostgresTimelineRepository) UpdateTimeline(ctx context.Context, t *domain.Timeline) error {
	query := `
		UPDATE timelines
		SET name = $1, updated_at = $2
		WHERE id = $3;
	`
	res, err := r.db.ExecContext(ctx, query, t.Name, t.UpdatedAt, t.ID)
	if err != nil {
		return fmt.Errorf("failed to update timeline: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrTimelineNotFound
	}
	return nil
}

func (r *PostgresTimelineRepository) DeleteTimeline(ctx context.Context, id string) error {
	query := `DELETE FROM timelines WHERE id = $1;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete timeline: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrTimelineNotFound
	}
	return nil
}

// --- Tracks ---

func (r *PostgresTimelineRepository) CreateTrack(ctx context.Context, tr *domain.Track) error {
	query := `
		INSERT INTO tracks (id, timeline_id, type, name, track_order, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7);
	`
	_, err := r.db.ExecContext(ctx, query, tr.ID, tr.TimelineID, string(tr.Type), tr.Name, tr.TrackOrder, tr.CreatedAt, tr.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert track: %w", err)
	}
	return nil
}

func (r *PostgresTimelineRepository) GetTrackByID(ctx context.Context, id string) (*domain.Track, error) {
	query := `
		SELECT id, timeline_id, type, name, track_order, created_at, updated_at
		FROM tracks
		WHERE id = $1;
	`
	var tr domain.Track
	var typeStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tr.ID, &tr.TimelineID, &typeStr, &tr.Name, &tr.TrackOrder, &tr.CreatedAt, &tr.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTrackNotFound
		}
		return nil, fmt.Errorf("failed to get track by id: %w", err)
	}
	tr.Type = domain.TrackType(typeStr)
	return &tr, nil
}

func (r *PostgresTimelineRepository) ListTracksByTimelineID(ctx context.Context, timelineID string) ([]*domain.Track, error) {
	query := `
		SELECT id, timeline_id, type, name, track_order, created_at, updated_at
		FROM tracks
		WHERE timeline_id = $1
		ORDER BY track_order ASC, created_at ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, timelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks by timeline id: %w", err)
	}
	defer rows.Close()

	var tracks []*domain.Track
	for rows.Next() {
		var tr domain.Track
		var typeStr string
		err := rows.Scan(
			&tr.ID, &tr.TimelineID, &typeStr, &tr.Name, &tr.TrackOrder, &tr.CreatedAt, &tr.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan track row: %w", err)
		}
		tr.Type = domain.TrackType(typeStr)
		tracks = append(tracks, &tr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("track row iteration error: %w", err)
	}

	if tracks == nil {
		tracks = []*domain.Track{}
	}
	return tracks, nil
}

func (r *PostgresTimelineRepository) UpdateTrack(ctx context.Context, tr *domain.Track) error {
	query := `
		UPDATE tracks
		SET name = $1, track_order = $2, updated_at = $3
		WHERE id = $4;
	`
	res, err := r.db.ExecContext(ctx, query, tr.Name, tr.TrackOrder, tr.UpdatedAt, tr.ID)
	if err != nil {
		return fmt.Errorf("failed to update track: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrTrackNotFound
	}
	return nil
}

func (r *PostgresTimelineRepository) DeleteTrack(ctx context.Context, id string) error {
	query := `DELETE FROM tracks WHERE id = $1;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete track: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrTrackNotFound
	}
	return nil
}

// --- Clips ---

func (r *PostgresTimelineRepository) CreateClip(ctx context.Context, c *domain.TimelineClip) error {
	query := `
		INSERT INTO timeline_clips (
			id, track_id, media_asset_id, timeline_start, source_in, source_out, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8);
	`
	_, err := r.db.ExecContext(
		ctx, query,
		c.ID, c.TrackID, c.MediaAssetID, c.TimelineStart, c.SourceIn, c.SourceOut, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert timeline clip: %w", err)
	}
	c.ComputeTimings()
	return nil
}

func (r *PostgresTimelineRepository) GetClipByID(ctx context.Context, id string) (*domain.TimelineClip, error) {
	query := `
		SELECT
			c.id, c.track_id, c.media_asset_id, c.timeline_start, c.source_in, c.source_out, c.created_at, c.updated_at,
			m.id, m.project_id, m.original_object_key, m.proxy_object_key, m.thumbnail_object_key,
			m.original_filename, m.mime_type, m.file_size, m.duration, m.width, m.height, m.fps,
			m.status, m.error_message, m.created_at, m.updated_at
		FROM timeline_clips c
		LEFT JOIN media_assets m ON c.media_asset_id = m.id
		WHERE c.id = $1;
	`
	var c domain.TimelineClip
	var m domain.MediaAsset
	var statusStr string
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&c.ID, &c.TrackID, &c.MediaAssetID, &c.TimelineStart, &c.SourceIn, &c.SourceOut, &c.CreatedAt, &c.UpdatedAt,
		&m.ID, &m.ProjectID, &m.OriginalObjectKey, &m.ProxyObjectKey, &m.ThumbnailObjectKey,
		&m.OriginalFilename, &m.MimeType, &m.FileSize, &m.Duration, &m.Width, &m.Height, &m.FPS,
		&statusStr, &m.ErrorMessage, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrClipNotFound
		}
		return nil, fmt.Errorf("failed to get timeline clip by id: %w", err)
	}
	m.Status = domain.MediaStatus(statusStr)
	c.Media = &m
	c.ComputeTimings()
	return &c, nil
}

func (r *PostgresTimelineRepository) ListClipsByTrackID(ctx context.Context, trackID string) ([]*domain.TimelineClip, error) {
	query := `
		SELECT
			c.id, c.track_id, c.media_asset_id, c.timeline_start, c.source_in, c.source_out, c.created_at, c.updated_at,
			m.id, m.project_id, m.original_object_key, m.proxy_object_key, m.thumbnail_object_key,
			m.original_filename, m.mime_type, m.file_size, m.duration, m.width, m.height, m.fps,
			m.status, m.error_message, m.created_at, m.updated_at
		FROM timeline_clips c
		LEFT JOIN media_assets m ON c.media_asset_id = m.id
		WHERE c.track_id = $1
		ORDER BY c.timeline_start ASC, c.created_at ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clips by track id: %w", err)
	}
	defer rows.Close()

	var clips []*domain.TimelineClip
	for rows.Next() {
		var c domain.TimelineClip
		var m domain.MediaAsset
		var statusStr string
		err := rows.Scan(
			&c.ID, &c.TrackID, &c.MediaAssetID, &c.TimelineStart, &c.SourceIn, &c.SourceOut, &c.CreatedAt, &c.UpdatedAt,
			&m.ID, &m.ProjectID, &m.OriginalObjectKey, &m.ProxyObjectKey, &m.ThumbnailObjectKey,
			&m.OriginalFilename, &m.MimeType, &m.FileSize, &m.Duration, &m.Width, &m.Height, &m.FPS,
			&statusStr, &m.ErrorMessage, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeline clip row: %w", err)
		}
		m.Status = domain.MediaStatus(statusStr)
		c.Media = &m
		c.ComputeTimings()
		clips = append(clips, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clip row iteration error: %w", err)
	}

	if clips == nil {
		clips = []*domain.TimelineClip{}
	}
	return clips, nil
}

func (r *PostgresTimelineRepository) ListClipsByTimelineID(ctx context.Context, timelineID string) ([]*domain.TimelineClip, error) {
	query := `
		SELECT
			c.id, c.track_id, c.media_asset_id, c.timeline_start, c.source_in, c.source_out, c.created_at, c.updated_at,
			m.id, m.project_id, m.original_object_key, m.proxy_object_key, m.thumbnail_object_key,
			m.original_filename, m.mime_type, m.file_size, m.duration, m.width, m.height, m.fps,
			m.status, m.error_message, m.created_at, m.updated_at
		FROM timeline_clips c
		INNER JOIN tracks tr ON c.track_id = tr.id
		LEFT JOIN media_assets m ON c.media_asset_id = m.id
		WHERE tr.timeline_id = $1
		ORDER BY tr.track_order ASC, c.timeline_start ASC, c.created_at ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, timelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to list clips by timeline id: %w", err)
	}
	defer rows.Close()

	var clips []*domain.TimelineClip
	for rows.Next() {
		var c domain.TimelineClip
		var m domain.MediaAsset
		var statusStr string
		err := rows.Scan(
			&c.ID, &c.TrackID, &c.MediaAssetID, &c.TimelineStart, &c.SourceIn, &c.SourceOut, &c.CreatedAt, &c.UpdatedAt,
			&m.ID, &m.ProjectID, &m.OriginalObjectKey, &m.ProxyObjectKey, &m.ThumbnailObjectKey,
			&m.OriginalFilename, &m.MimeType, &m.FileSize, &m.Duration, &m.Width, &m.Height, &m.FPS,
			&statusStr, &m.ErrorMessage, &m.CreatedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeline clip row: %w", err)
		}
		m.Status = domain.MediaStatus(statusStr)
		c.Media = &m
		c.ComputeTimings()
		clips = append(clips, &c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("clip row iteration error: %w", err)
	}

	if clips == nil {
		clips = []*domain.TimelineClip{}
	}
	return clips, nil
}

func (r *PostgresTimelineRepository) UpdateClip(ctx context.Context, c *domain.TimelineClip) error {
	query := `
		UPDATE timeline_clips
		SET timeline_start = $1, source_in = $2, source_out = $3, updated_at = $4
		WHERE id = $5;
	`
	res, err := r.db.ExecContext(ctx, query, c.TimelineStart, c.SourceIn, c.SourceOut, c.UpdatedAt, c.ID)
	if err != nil {
		return fmt.Errorf("failed to update timeline clip: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrClipNotFound
	}
	c.ComputeTimings()
	return nil
}

func (r *PostgresTimelineRepository) DeleteClip(ctx context.Context, id string) error {
	query := `DELETE FROM timeline_clips WHERE id = $1;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete timeline clip: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrClipNotFound
	}
	return nil
}

// --- Full Hierarchy ---

func (r *PostgresTimelineRepository) GetFullTimeline(ctx context.Context, projectID string) (*domain.Timeline, error) {
	timeline, err := r.GetTimelineByProjectID(ctx, projectID)
	if err != nil {
		return nil, err
	}

	tracks, err := r.ListTracksByTimelineID(ctx, timeline.ID)
	if err != nil {
		return nil, err
	}

	clips, err := r.ListClipsByTimelineID(ctx, timeline.ID)
	if err != nil {
		return nil, err
	}

	// Map clips to tracks
	clipsByTrack := make(map[string][]*domain.TimelineClip)
	for _, c := range clips {
		clipsByTrack[c.TrackID] = append(clipsByTrack[c.TrackID], c)
	}

	for _, tr := range tracks {
		if trClips, ok := clipsByTrack[tr.ID]; ok {
			tr.Clips = trClips
		} else {
			tr.Clips = []*domain.TimelineClip{}
		}
	}

	timeline.Tracks = tracks
	return timeline, nil
}
