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
- **Role:** Presents user interfaces for project management, media ingestion, film storyboards, timeline editing, and playback.
- **Boundaries:**
  - Connects to the Go API (`VITE_API_URL`) for metadata, project operations, and timeline state.
  - Uploads raw media files directly to the storage provider via presigned URLs obtained from the Go API.
  - Plays back 720p proxy video streams directly from decorated storage URLs.
  - Never calls Python AI services directly, and never connects directly to databases.

### 2. Core API Backend (`apps/api`)
- **Technology:** Idiomatic Go standard library (`net/http`), `database/sql`, `github.com/lib/pq`, `github.com/google/uuid`.
- **Role:** System of record coordinator, REST API gateway, media ingestion orchestrator, timeline state manager, and database manager.
- **Structure:**
  - `cmd/server/main.go`: Application bootstrap, database connection pooling, auto-migration trigger, and server lifecycle.
  - `internal/config/`: Environment configuration loading (`PORT`, `DATABASE_URL`, `STORAGE_DRIVER`, `STORAGE_BASE_URL`, `FFMPEG_PATH`, `FFPROBE_PATH`).
  - `internal/domain/`: Domain entities and request validation logic (`Project`, `MediaAsset`, `Timeline`, `Track`, `TimelineClip`).
  - `internal/repository/`: Abstract persistence interfaces (`ProjectRepository`, `MediaRepository`, `TimelineRepository`) and PostgreSQL implementations.
  - `internal/service/`: Business rules, validation, timestamp orchestration, and UUID generation (`ProjectService`, `MediaService`, `TimelineService`).
  - `internal/storage/`: Object storage abstraction (`Storage` interface, `LocalStorage`, `key.go` validation).
  - `internal/media/`: Video processing (`FFmpegProcessor` for ffprobe inspection, proxy transcoding, and thumbnail extraction) and background dispatching (`media.Queue` interface and `MemoryQueue`).
  - `internal/handlers/`: HTTP request dispatching, JSON decoding, status code management, local storage file serving, stream limits (5GB MaxBytesReader), and structured error responses.
  - `internal/database/migrations/`: Embeddable transactional SQL migration runner.

---

## Timeline Architecture (Hop 3A)

The timeline model represents the canonical non-linear editing (NLE) state that both the human interface and future AI editing systems operate on:

```text
Timeline (1 per project)
 └── Tracks (ordered by track_order: 0, 1, 2...)
      ├── Video Tracks (VIDEO)
      └── Audio Tracks (AUDIO)
           └── TimelineClips (placed at timeline_start)
                └── Media Asset Reference (source_in → source_out)
```

### 1. Integer Microsecond Timebase
- Editing state uses integer microseconds (`int64`, $1\text{s} = 1,000,000\mu\text{s}$).
- Avoids floating-point precision loss across common film and broadcast frame rates (23.976, 24, 25, 29.97, 30, 50, 60 fps).

### 2. Clip Semantics & Single Source of Truth
- A `TimelineClip` defines:
  - `timeline_start`: Position where the clip begins on the timeline track (in microseconds).
  - `source_in`: In-point of the source media asset (in microseconds).
  - `source_out`: Out-point of the source media asset (in microseconds).
- Duration is computed on-the-fly (`source_out - source_in`), and timeline end is computed (`timeline_start + duration`). Redundant duration columns are omitted from storage to prevent state divergence.

### 3. Track Overlap Policy
- **Same Track:** Clips on the same track **must not overlap**. Any attempt to insert or move a clip such that it intersects with an existing clip on the same track returns `ErrClipOverlap` (HTTP 409 Conflict). Abutting clips (where `clipA.end == clipB.start`) are explicitly allowed.
- **Different Tracks:** Overlap is allowed across separate tracks (e.g. B-roll overlapping on Video 2 over Video 1, or multiple audio layers playing simultaneously).

### 4. Cross-Project Ownership Isolation
- Every timeline, track, and clip request strictly validates project ownership:
  - Timeline must belong to the specified project.
  - Track must belong to the specified timeline.
  - Clip must belong to the specified track.
  - Media asset referenced by a clip must belong to the exact same project (`media.ProjectID == project.ID`). Cross-project media leakage is rejected at the domain level.

---

## Media Ingestion Pipeline (Hop 2)
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

---

## Architectural Decisions Log

1. **Integer Microsecond Editing Representation:** Standardized all timeline positions and in/out points on integer microseconds (`int64`).
2. **Canonical Non-Overlapping Track Model:** Enforced strict non-overlap per track at the service layer while permitting multi-track layering.
3. **Single Timeline per Project for Initial Hops:** Maintained 1-to-1 project-to-timeline mapping without premature sequence nesting complexity.
4. **Direct-to-Storage Presigned Uploads:** Raw video files are uploaded directly to the storage provider URL with a 5GB maximum body stream limit.
5. **Local-Only Storage for Hop 2/3A:** `LocalStorage` is the only supported driver for this hop, with strict path traversal confinement and clean error handling.
6. **Deterministic Queue Enqueueing:** `CompleteUpload` only transitions the database record to `PROCESSING` if enqueueing succeeds; if enqueueing fails, the asset is saved as `FAILED` and an error is returned.
7. **Retry Semantics:** Failed media assets can be retried via `CompleteUpload` (transitioning `FAILED` -> `PROCESSING`), and `ProcessMedia` is idempotent on `READY` assets.
