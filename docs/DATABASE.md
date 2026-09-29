# AI Filmmaker - Database & Schema Documentation

## Database Engine
- **Primary Database:** PostgreSQL 16
- **Connection Strategy:** Required startup dependency. The Go API terminates immediately with a fatal exit code if PostgreSQL is unreachable.
- **Connection Pool:** `database/sql` via `github.com/lib/pq` with configured connection limits:
  - Max Open Connections: 25
  - Max Idle Connections: 10
  - Connection Max Lifetime: 5 minutes

---

## Migration Ownership & Execution

The Go embedded migration runner (`apps/api/internal/database/migrations/`) is the **single authoritative owner** of all database schema migrations across all environments (local development, CI, production):

1. **Storage:** SQL migration files reside solely in `apps/api/internal/database/migrations/sql/` and are embedded into the Go binary via `//go:embed`.
2. **Execution Timing:** During API startup, immediately following a verified database connection, the migration runner acquires a connection and scans `schema_migrations`.
3. **Idempotency & Safety:** Pending migrations are executed in ascending version order, each enclosed within an isolated transaction (`BEGIN ... COMMIT`). Once committed, the version is recorded in `schema_migrations`. Already applied migrations are skipped cleanly without re-execution.
4. **Failure Behavior:** If any migration fails, the transaction rolls back, an error is logged, and the Go API process exits with a fatal status code prior to opening HTTP ports.
5. **Docker Compose Role:** Docker Compose is strictly an infrastructure provider (PostgreSQL & Redis). It contains no custom initialization SQL scripts.

---

## Schema Definitions

### `projects` Table (Hop 1)

Represents top-level filmmaking productions.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `VARCHAR(36)` | `PRIMARY KEY` | UUID string identifier |
| `name` | `VARCHAR(255)` | `NOT NULL` | Project / film title |
| `description` | `TEXT` | `NOT NULL DEFAULT ''` | Project logline or synopsis |
| `created_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Project creation timestamp |
| `updated_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Last updated timestamp |

#### Indexes
- `idx_projects_created_at` on `projects(created_at DESC)` for efficient reverse-chronological project listing.

---

### `media_assets` Table (Hop 2)

Represents video takes, footage, audio, and media files ingested into a project.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `VARCHAR(36)` | `PRIMARY KEY` | Media asset UUID string |
| `project_id` | `VARCHAR(36)` | `NOT NULL REFERENCES projects(id) ON DELETE CASCADE` | Foreign key linking media to parent project |
| `original_object_key` | `TEXT` | `NOT NULL` | Storage object key for the raw uploaded file |
| `proxy_object_key` | `TEXT` | `NULL` | Storage object key for 720p H.264 browser proxy |
| `thumbnail_object_key` | `TEXT` | `NULL` | Storage object key for generated JPEG thumbnail |
| `original_filename` | `VARCHAR(255)` | `NOT NULL` | Original sanitized file name |
| `mime_type` | `VARCHAR(100)` | `NOT NULL` | MIME type (e.g. `video/mp4`, `video/quicktime`) |
| `file_size` | `BIGINT` | `NOT NULL DEFAULT 0` | File size in bytes |
| `duration` | `DOUBLE PRECISION` | `NOT NULL DEFAULT 0` | Extracted duration in seconds |
| `width` | `INTEGER` | `NOT NULL DEFAULT 0` | Video width in pixels |
| `height` | `INTEGER` | `NOT NULL DEFAULT 0` | Video height in pixels |
| `fps` | `DOUBLE PRECISION` | `NOT NULL DEFAULT 0` | Video frames per second |
| `status` | `VARCHAR(50)` | `NOT NULL DEFAULT 'UPLOADING'` | Lifecycle status (`UPLOADING`, `PROCESSING`, `READY`, `FAILED`) |
| `error_message` | `TEXT` | `NULL` | Error details if status is `FAILED` |
| `created_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |
| `updated_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Last updated timestamp |

#### Indexes & Constraints
- Foreign key: `fk_media_assets_project_id` references `projects(id)` with `ON DELETE CASCADE`.
- Index: `idx_media_assets_project_id` on `media_assets(project_id)` for rapid project asset queries.
- Index: `idx_media_assets_status` on `media_assets(status)` for worker queue querying and startup reconciliation.
- Index: `idx_media_assets_created_at` on `media_assets(created_at DESC)` for reverse chronological display.

---

### `timelines` Table (Hop 3A)

Represents a project's editing timeline sequence.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `VARCHAR(36)` | `PRIMARY KEY` | Timeline UUID string |
| `project_id` | `VARCHAR(36)` | `NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE` | 1-to-1 project ownership relationship |
| `name` | `VARCHAR(255)` | `NOT NULL DEFAULT 'Main Timeline'` | Sequence display name |
| `created_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Creation timestamp |
| `updated_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Update timestamp |

#### Indexes & Constraints
- Unique key: `UNIQUE(project_id)` enforcing single primary timeline per project for initial hops.
- Index: `idx_timelines_project_id` on `timelines(project_id)`.

---

### `tracks` Table (Hop 3A)

Represents ordered audio/video layers within a timeline.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `VARCHAR(36)` | `PRIMARY KEY` | Track UUID string |
| `timeline_id` | `VARCHAR(36)` | `NOT NULL REFERENCES timelines(id) ON DELETE CASCADE` | Parent timeline foreign key |
| `type` | `VARCHAR(32)` | `NOT NULL` | Track type (`VIDEO` or `AUDIO`) |
| `name` | `VARCHAR(255)` | `NOT NULL` | Track label (e.g. "Video 1", "Dialogue") |
| `track_order` | `INTEGER` | `NOT NULL DEFAULT 0` | Explicit ordering index (0-indexed) |
| `created_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Creation timestamp |
| `updated_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Update timestamp |

#### Indexes & Constraints
- Index: `idx_tracks_timeline_id` on `tracks(timeline_id, track_order ASC)`.

---

### `timeline_clips` Table (Hop 3A)

Represents placed slices of media assets positioned on a track.

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | `VARCHAR(36)` | `PRIMARY KEY` | Clip UUID string |
| `track_id` | `VARCHAR(36)` | `NOT NULL REFERENCES tracks(id) ON DELETE CASCADE` | Parent track foreign key |
| `media_asset_id` | `VARCHAR(36)` | `NOT NULL REFERENCES media_assets(id) ON DELETE CASCADE` | Referenced media asset foreign key |
| `timeline_start` | `BIGINT` | `NOT NULL` | Timeline placement in microseconds |
| `source_in` | `BIGINT` | `NOT NULL DEFAULT 0` | Media in-point in microseconds |
| `source_out` | `BIGINT` | `NOT NULL` | Media out-point in microseconds |
| `created_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Creation timestamp |
| `updated_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Update timestamp |

#### Timing Notes & Invariants
- `duration` is computed as `source_out - source_in`. Redundant duration columns are intentionally avoided to maintain single-source-of-truth consistency.
- `timeline_end` is computed as `timeline_start + duration`.
- Timebase: **Integer Microseconds** (`1 second = 1,000,000 microseconds`).

#### Indexes & Constraints
- Index: `idx_timeline_clips_track_id` on `timeline_clips(track_id, timeline_start ASC)`.
- Index: `idx_timeline_clips_media_asset_id` on `timeline_clips(media_asset_id)`.

---

### `schema_migrations` Table

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `version` | `VARCHAR(255)` | `PRIMARY KEY` | Migration version identifier (e.g. `000001`, `000002`, `000003`) |
| `name` | `VARCHAR(255)` | `NOT NULL` | Migration file name |
| `applied_at` | `TIMESTAMP WITH TIME ZONE` | `DEFAULT CURRENT_TIMESTAMP` | Timestamp when migration was committed |
