# AI Filmmaker - Architecture Documentation

## System Overview

AI Filmmaker is an AI-native filmmaking platform designed around strict subsystem boundaries:

```text
React / TypeScript (apps/web)
        ↓ (REST API / Storage Upload)
      Go API (apps/api)
        ↓
 PostgreSQL / Local/S3 Storage / Background Worker (infrastructure)
        ↓ (HTTP / gRPC)
 Python AI services (apps/ai)
```

---

## Core Layers & Responsibilities

### 1. Frontend (`apps/web`)
- **Technology:** React 18, TypeScript, Vite.
- **Role:** Presents user interfaces for project management, media ingestion, film storyboards, and editor views.
- **Boundaries:**
  - Connects to the Go API (`VITE_API_URL`) for all metadata and upload coordination.
  - Uploads raw media files directly to the storage provider via presigned URLs obtained from the Go API.
  - Plays back 720p proxy video streams directly from decorated storage URLs.
  - Never calls Python AI services directly, and never connects directly to databases.

### 2. Core API Backend (`apps/api`)
- **Technology:** Idiomatic Go standard library (`net/http`), `database/sql`, `github.com/lib/pq`, `github.com/google/uuid`.
- **Role:** System of record coordinator, REST API gateway, media ingestion orchestrator, and database manager.
- **Structure:**
  - `cmd/server/main.go`: Application bootstrap, database connection pooling, auto-migration trigger, and server lifecycle.
  - `internal/config/`: Environment configuration loading (`PORT`, `DATABASE_URL`, `STORAGE_DRIVER`, `STORAGE_BASE_URL`, `FFMPEG_PATH`, `FFPROBE_PATH`).
  - `internal/domain/`: Domain entities and request validation logic (`Project`, `MediaAsset`, `MediaStatus`, `MediaMetadata`).
  - `internal/repository/`: Abstract persistence interfaces (`ProjectRepository`, `MediaRepository`) and PostgreSQL implementations.
  - `internal/service/`: Business rules, validation, timestamp orchestration, and UUID generation (`ProjectService`, `MediaService`).
  - `internal/storage/`: Object storage abstraction (`Storage` interface, `LocalStorage`, `S3Storage`).
  - `internal/media/`: Video processing (`FFmpegProcessor` for ffprobe inspection, proxy transcoding, and thumbnail extraction) and background dispatching (`media.Queue` interface and `MemoryQueue`).
  - `internal/handlers/`: HTTP request dispatching, JSON decoding, status code management, local storage file serving, and structured error responses.
  - `internal/database/migrations/`: Embeddable transactional SQL migration runner.

### 3. Media Ingestion Pipeline (Hop 2)
```text
Client Browser
   │ 1. POST /projects/:id/media/upload
   ▼
Go API (Validates request, creates MediaAsset with status UPLOADING, generates presigned PUT URL)
   │ 2. Returns { uploadUrl, mediaId, objectKey }
   ▼
Client Browser
   │ 3. PUT raw video directly to Storage URL
   ▼
Object Storage (Stores raw media at projects/:projectId/media/raw/:objectKey)
   │
Client Browser
   │ 4. POST /projects/:id/media/:id/complete
   ▼
Go API (Verifies file existence in storage, sets status to PROCESSING, enqueues to Worker Queue)
   │ 5. Asynchronous Background Job
   ▼
Worker (FFmpeg / FFprobe)
   ├── A. Inspect metadata (ffprobe: duration, width, height, fps, codec)
   ├── B. Generate 720p H.264 / AAC MP4 proxy (scale to 1280x720, faststart)
   ├── C. Generate JPEG thumbnail (at 1.0s or mid-point)
   ├── D. Upload proxy and thumbnail to Object Storage
   └── E. Update MediaAsset in PostgreSQL (status = READY, object keys, dimensions, duration, fps)
```

### 4. Background Processing Boundary
- An explicit `media.Queue` interface separates the HTTP handler lifecycle from the CPU-heavy transcoding jobs.
- For Hop 2, a thread-safe, bounded in-memory worker queue is used without requiring Kafka or RabbitMQ.
- In future enterprise production hops, this interface will be swapped directly to Temporal workflows without altering the Go domain or service layers.

### 5. Python AI Service (`apps/ai`)
- **Technology:** Python 3.10+, FastAPI, `uv`.
- **Role:** AI/ML workload orchestration, model inference, and Film DSL generation.
- **Boundaries:** Only reachable by internal Go backend calls (via HTTP/gRPC).

---

## Architectural Decisions Log

1. **Direct-to-Storage Presigned Uploads:** Raw video files are never proxied or streamed through the Go API web server memory, eliminating backend bandwidth bottlenecks.
2. **Storage Abstraction:** The `Storage` interface allows local filesystem operations for dev/testing (`LocalStorage`) and S3-compatible cloud buckets in production without code changes.
3. **Deterministic Object Key Isolation:** Media files are partitioned under `projects/<projectId>/media/{raw,proxies,thumbnails}/<filename>`.
4. **Server-Side Metadata Enforcement:** Client-provided dimensions, FPS, or durations are never trusted. The server runs `ffprobe` to derive definitive technical parameters.
5. **Idempotent Retry Safety:** If media processing fails, it marks status as `FAILED` with an error message; re-processing can safely overwrite proxy and thumbnail object keys deterministically without corrupting database records.
