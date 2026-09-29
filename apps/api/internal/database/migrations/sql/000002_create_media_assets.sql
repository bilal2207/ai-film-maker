-- Hop 2: Create media_assets table
CREATE TABLE IF NOT EXISTS media_assets (
    id VARCHAR(36) PRIMARY KEY,
    project_id VARCHAR(36) NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    original_object_key TEXT NOT NULL,
    proxy_object_key TEXT,
    thumbnail_object_key TEXT,
    original_filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    duration DOUBLE PRECISION NOT NULL DEFAULT 0,
    width INTEGER NOT NULL DEFAULT 0,
    height INTEGER NOT NULL DEFAULT 0,
    fps DOUBLE PRECISION NOT NULL DEFAULT 0,
    status VARCHAR(50) NOT NULL DEFAULT 'UPLOADING',
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_media_assets_project_id ON media_assets(project_id);
CREATE INDEX IF NOT EXISTS idx_media_assets_status ON media_assets(status);
CREATE INDEX IF NOT EXISTS idx_media_assets_created_at ON media_assets(created_at DESC);
