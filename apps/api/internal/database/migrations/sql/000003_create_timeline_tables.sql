-- Hop 3A: Create timeline, tracks, and timeline_clips tables

CREATE TABLE IF NOT EXISTS timelines (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT 'Main Timeline',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_timelines_project_id ON timelines(project_id);

CREATE TABLE IF NOT EXISTS tracks (
    id VARCHAR(36) PRIMARY KEY,
    timeline_id VARCHAR(36) NOT NULL REFERENCES timelines(id) ON DELETE CASCADE,
    type VARCHAR(32) NOT NULL,
    name VARCHAR(255) NOT NULL,
    track_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tracks_timeline_id ON tracks(timeline_id, track_order ASC);

CREATE TABLE IF NOT EXISTS timeline_clips (
    id VARCHAR(36) PRIMARY KEY,
    track_id VARCHAR(36) NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    media_asset_id VARCHAR(36) NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE,
    timeline_start BIGINT NOT NULL,
    source_in BIGINT NOT NULL DEFAULT 0,
    source_out BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_timeline_clips_track_id ON timeline_clips(track_id, timeline_start ASC);
CREATE INDEX IF NOT EXISTS idx_timeline_clips_media_asset_id ON timeline_clips(media_asset_id);
