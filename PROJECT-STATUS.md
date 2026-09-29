# AI Filmmaker - Project Status

## Current Hop: Hop 2 (Media Ingestion)
**Status:** COMPLETED & VERIFIED (Reliability & Security Hardened)

---

### Vision
AI-native filmmaking platform spanning the complete production lifecycle:
`Idea → Script → Production → Footage → Edit → VFX → Color → Sound → Distribution`.

---

### Hop 2 Objectives & Deliverables
- [x] Database schema for media assets (`media_assets` table with `projects(id)` foreign key, cascading delete, and indexes on `project_id`, `status`, and `created_at DESC`).
- [x] Go domain model & lifecycle state machine (`UPLOADING → PROCESSING → READY` or `FAILED`).
- [x] Security and validation:
  - Centralized storage key validation rejecting absolute paths, `..` traversal, backslashes, null bytes, and Windows drive letters.
  - Strict base directory confinement in `LocalStorage` with automatic cleanup of partial uploads.
  - Stream-level 5GB body limit enforcement via `http.MaxBytesReader` in storage handlers.
  - MIME type whitelisting (`video/mp4`, `video/quicktime`, `video/webm`, `video/x-matroska`, etc.).
  - Filename sanitization (path traversal protection).
  - Server-side metadata extraction (client duration/resolution never trusted).
- [x] Storage Abstraction Layer (`apps/api/internal/storage`):
  - `LocalStorage` provider for zero-dependency local development with byte-range HTTP streaming.
  - `storage.NewFromConfig` explicitly rejects unsupported drivers (`"s3"`) in Hop 2 to maintain truth in implementation.
- [x] FFmpeg / FFprobe Media Processing Pipeline:
  - Portable binary discovery in standard system paths and user tool directories (`.tools/ffmpeg/bin`).
  - Strict JSON `ffprobe` metadata inspection (duration, width, height, FPS, codec).
  - 720p H.264 / AAC proxy transcoding for web and editor preview.
  - Frame extraction for JPEG thumbnail generation.
- [x] Background Processing & Queue Boundary:
  - Clean `media.Queue` interface decoupling heavy transcoding from the HTTP request cycle.
  - Bounded in-memory worker queue with graceful shutdown and worker pooling.
  - Deterministic queue failure handling: if `queue.Enqueue` fails, the asset transitions to `FAILED` with an error message and does not remain stuck in `PROCESSING`.
  - Startup reconciliation: `ReconcileOrphanedProcessing` scans for orphaned `PROCESSING` records on server startup and marks them as `FAILED` with retry guidance.
  - Idempotent processing: repeated processing of `READY` assets is a no-op.
- [x] Complete REST API endpoints:
  - `POST /projects/:projectId/media/upload` (Initialize upload & presigned URL)
  - `POST /projects/:projectId/media/:mediaId/complete` (Finalize upload & enqueue processing, supports retrying failed assets)
  - `GET /projects/:projectId/media` (List project footage)
  - `GET /projects/:projectId/media/:mediaId` (Get asset metadata & playback URLs)
  - `DELETE /projects/:projectId/media/:mediaId` (Delete media & storage objects)
- [x] Frontend Media Ingestion:
  - React media API client (`apps/web/src/api/media.ts`).
  - Direct-to-storage upload with progress percentage tracking.
  - Media asset cards with thumbnail images, status badges (`READY`, `PROCESSING`, `UPLOADING`, `FAILED`), specs, and delete action.
  - Automatic polling when jobs are pending/processing.
  - 720p proxy video player preview modal.
- [x] Comprehensive Testing:
  - Go domain, storage traversal, factory, repository, queue failure, startup reconciliation, idempotency, and handler tests.
  - Vitest frontend component tests for upload flow, states, modal, and deletion.
  - Package builds for `film-dsl` and `editor-core`.
  - Python AI linting (`ruff`) and test suite (`pytest`).

---

### Previous Deliverables
- **Hop 0 / 0.1:** Foundational Monorepo Setup, dependency boundaries, Go API, Python AI service, React frontend, `@ai-filmmaker/film-dsl`, `@ai-filmmaker/editor-core`.
- **Hop 1:** Project System, PostgreSQL migrations, project CRUD endpoints, frontend Project List and Project Shell.

---

### Intentional Limitations & Deferred Features
1. **Authentication / Authorization:** Multi-tenant user auth (Clerk) remains deferred; media assets belong to projects without user-based access control.
2. **Queue Durability:** `MemoryQueue` is in-memory; process restarts interrupt in-flight jobs, which are reconciled to `FAILED` on startup so users can retry. Replacing this with Temporal is deferred to future enterprise hops.
3. **Cloud S3 Adapter:** Cloud S3 integration is explicitly unsupported in Hop 2; only local storage is supported.
4. **Timeline / NLE / WebGPU:** Media assets are ingested and proxied; timeline sequencing, clip trimming, and WebGPU canvas rendering are deferred to upcoming editor hops.
5. **AI Generation / Vision Analysis:** Scene detection, shot tagging, Whisper speech-to-text, and embeddings remain deferred to AI orchestration hops.

---

### Next Hop: Hop 3
- Script & Storyboard Data Models
- Scene and Shot Decomposition API
- Film DSL Schema Generation Foundation
