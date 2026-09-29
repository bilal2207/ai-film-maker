# Migrations

All database migrations are managed and applied solely by the Go API migration runner (`apps/api/internal/database/migrations/`).

Migration SQL files are stored in:
`apps/api/internal/database/migrations/sql/`

Docker Compose runs standard PostgreSQL instances without custom initialization SQL scripts.
