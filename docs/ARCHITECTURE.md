# AI Filmmaker - Architecture Documentation

## System Overview

AI Filmmaker is an AI-native filmmaking platform designed around strict subsystem boundaries:

```text
React / TypeScript (apps/web)
        ↓ (REST API / Direct Storage Upload)
      Go API (apps/api)
        ↓
 PostgreSQL / Local Storage / Background Worker (infrastructure)
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
  - `internal/storage/`: Object storage abstraction (`Storage` interface, `LocalStorage`, `key.go` validation).
  - `internal/media/`: Video processing (`FFmpegProcessor` for ffprobe inspection, proxy transcoding, and thumbnail extraction) and background dispatching (`media.Queue` interface and `MemoryQueue`).
  - `internal/handlers/`: HTTP request dispatching, JSON decoding, status code management, local storage file serving, stream limits (5GB MaxBytesReader), and structured error responses.
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
Object Storage (Stores raw media under projects/:projectId/media/:mediaId/original/:filename)
   │
Client Browser
   │ 4. POST /projects/:id/media/:id/complete
   ▼
Go API (Verifies file existence in storage, enqueues to Worker Queue, then sets status to PROCESSING)
   │ 5. Asynchronous Background Job
   ▼
Worker (FFmpeg / FFprobe)
   ├── A. Inspect metadata (ffprobe: duration, width, height, fps, codec)
   ├── B. Generate 720p H.264 / AAC MP4 proxy (scale to 1280x720, faststart)
   ├── C. Generate JPEG thumbnail (at 1.0s or mid-point)
   ├── D. Upload proxy and thumbnail to Object Storage
   └── E. Update MediaAsset in PostgreSQL (status = READY, object keys, dimensions, duration, fps)
```

### 4. Background Processing & Queue Boundary
- **Queue Interface:** Heavy transcoding jobs are decoupled behind the `media.Queue` interface.
- **Hop 2 In-Memory Worker Pool:** Bounded, thread-safe in-memory worker queue (`MemoryQueue`).
- **Restart / Crash Limitations & Startup Reconciliation:**
  - `MemoryQueue` is non-durable across process restarts and crashes. In-flight jobs can be interrupted by a process restart.
  - On API startup, `ReconcileOrphanedProcessing` automatically scans for any assets left in `PROCESSING` status from a previous crashed run and marks them as `FAILED` with a diagnostic error message (`"processing interrupted by server restart; retry available"`).
  - This ensures assets are never permanently stuck in `PROCESSING`.
  - In a future enterprise hop, the `media.Queue` interface will be backed by Temporal without modifying service/domain layers.

### 5. Storage Driver & Path Traversal Security
- **Supported Driver in Hop 2:** `local` (`LocalStorage`) is the sole supported driver in Hop 2. The factory explicitly rejects `"s3"` or undefined external drivers with an unsupported driver error to prevent fake implementations.
- **Centralized Key Validation:** All storage keys are validated against strict whitelist patterns (`projects/{projectId}/media/{...}`). Absolute paths, Windows drive letters, null bytes, backslashes, and `..` traversal sequences are strictly rejected.
- **Base Directory Confinement:** Local filesystem paths are resolved and verified with `filepath.Rel` to guarantee they remain strictly inside the configured `baseDir`.
- **Stream Limit Enforcement:** Storage upload endpoints enforce a strict 5GB limit via `http.MaxBytesReader`. Partial files are deleted immediately if the stream aborts.

### 6. Python AI Service (`apps/ai`)
- **Technology:** Python 3.10+, FastAPI, `uv`.
- **Role:** AI/ML workload orchestration, model inference, and Film DSL generation.
- **Boundaries:** Only reachable by internal Go backend calls (via HTTP/gRPC).

---

## Architectural Decisions Log

1. **Direct-to-Storage Presigned Uploads:** Raw video files are uploaded directly to the storage provider URL with a 5GB maximum body stream limit.
2. **Local-Only Storage for Hop 2:** `LocalStorage` is the only supported driver for this hop, with strict path traversal confinement and clean error handling.
3. **Deterministic Object Key Isolation:** Media files are partitioned under `projects/<projectId>/media/<mediaId>/{original,proxy,thumbnails}/<filename>`.
4. **Server-Side Metadata Enforcement:** Client-provided dimensions, FPS, or durations are never trusted. The server runs `ffprobe` to derive definitive technical parameters.
5. **Deterministic Queue Enqueueing:** `CompleteUpload` only transitions the database record to `PROCESSING` if enqueueing succeeds; if enqueueing fails, the asset is saved as `FAILED` and an error is returned.
6. **Retry Semantics:** Failed media assets can be retried via `CompleteUpload` (transitioning `FAILED` -> `PROCESSING`), and `ProcessMedia` is idempotent on `READY` assets.
