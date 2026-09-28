-- Hop 0 Initial Migration Foundation
-- This sets up basic metadata tables for database schema versioning / health

CREATE TABLE IF NOT EXISTS schema_migrations (
    version VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO schema_migrations (version) VALUES ('000001_init')
ON CONFLICT (version) DO NOTHING;
