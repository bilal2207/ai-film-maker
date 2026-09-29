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
| `id` | `VARCHAR(36)` | `PRIMARY KEY` | Media asset UUID |
| `project_id` | `VARCHAR(36)` | `NOT NULL, REFERENCES projects(id) ON DELETE CASCADE` | Foreign key linking media to parent project |
| `original_object_key` | `VARCHAR(1024)` | `NOT NULL` | Storage object key for the raw uploaded file |
| `proxy_object_key` | `VARCHAR(1024)` | `NULL` | Storage object key for 720p H.264 browser proxy |
| `thumbnail_object_key` | `VARCHAR(1024)` | `NULL` | Storage object key for generated JPEG thumbnail |
| `original_filename` | `VARCHAR(512)` | `NOT NULL` | Original sanitized file name |
| `mime_type` | `VARCHAR(128)` | `NOT NULL` | MIME type (e.g. `video/mp4`, `video/quicktime`) |
| `file_size` | `BIGINT` | `NOT NULL DEFAULT 0` | File size in bytes |
| `duration` | `DOUBLE PRECISION` | `NULL` | Extracted duration in seconds |
| `width` | `INT` | `NULL` | Video width in pixels |
| `height` | `INT` | `NULL` | Video height in pixels |
| `fps` | `DOUBLE PRECISION` | `NULL` | Video frames per second |
| `status` | `VARCHAR(32)` | `NOT NULL DEFAULT 'UPLOADING'` | Lifecycle status (`UPLOADING`, `PROCESSING`, `READY`, `FAILED`) |
| `error_message` | `TEXT` | `NULL` | Error details if status is `FAILED` |
| `created_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Record creation timestamp |
| `updated_at` | `TIMESTAMP WITH TIME ZONE` | `NOT NULL DEFAULT CURRENT_TIMESTAMP` | Last updated timestamp |

#### Indexes & Constraints
- Foreign key: `fk_media_assets_project_id` references `projects(id)` with `ON DELETE CASCADE`.
- Index: `idx_media_assets_project_id` on `media_assets(project_id)` for rapid project asset queries.
- Index: `idx_media_assets_status` on `media_assets(status)` for worker queue querying.
- Index: `idx_media_assets_created_at` on `media_assets(created_at DESC)` for reverse chronological display.

---

### `schema_migrations` Table

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `version` | `VARCHAR(255)` | `PRIMARY KEY` | Migration version identifier (e.g. `000001`, `000002`) |
| `name` | `VARCHAR(255)` | `NOT NULL` | Migration file name |
| `applied_at` | `TIMESTAMP WITH TIME ZONE` | `DEFAULT CURRENT_TIMESTAMP` | Timestamp when migration was committed |
