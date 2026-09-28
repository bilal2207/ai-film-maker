# AI Filmmaker - Database & Schema Documentation

## Database Engine
- **Primary Database:** PostgreSQL 16
- **Connection Pool:** `database/sql` via `github.com/lib/pq` with configured connection limits:
  - Max Open Connections: 25
  - Max Idle Connections: 10
  - Connection Max Lifetime: 5 minutes

---

## Migration Strategy

Migrations are managed via the programmatic Go migration runner located at `apps/api/internal/database/migrations/`:
- **Storage:** SQL migration files reside in `apps/api/internal/database/migrations/sql/` and are embedded directly into the Go binary at compile time via `//go:embed`.
- **Tracking:** PostgreSQL table `schema_migrations` tracks applied versions and execution timestamps.
- **Execution:** When the Go API initializes, unapplied migrations are executed sequentially inside isolated database transactions (`BEGIN ... COMMIT`).

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

### `schema_migrations` Table

| Column | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `version` | `VARCHAR(255)` | `PRIMARY KEY` | Migration version identifier (e.g. `000001`) |
| `name` | `VARCHAR(255)` | `NOT NULL` | Migration file name |
| `applied_at` | `TIMESTAMP WITH TIME ZONE` | `DEFAULT CURRENT_TIMESTAMP` | Timestamp when migration was committed |
