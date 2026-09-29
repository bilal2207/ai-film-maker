# AI Filmmaker - Project Status

## Current Hop: Hop 3A (Basic NLE: Timeline Domain Model + Persistence)
**Status:** COMPLETED & VERIFIED

---

### Vision
AI-native filmmaking platform spanning the complete production lifecycle:
`Idea → Script → Production → Footage → Edit → VFX → Color → Sound → Distribution`.

---

### Hop 3A Objectives & Deliverables
- [x] Database schema for timeline system:
  - `timelines` table (1-to-1 project relationship, cascading delete).
  - `tracks` table (ordered layers, `VIDEO` / `AUDIO` track types).
  - `timeline_clips` table (referencing `tracks(id)` and `media_assets(id)` with foreign keys).
- [x] Canonical Timeline Domain Model (`apps/api/internal/domain/timeline.go`):
  - Integer microsecond timebase (`int64`, $1\text{s} = 1,000,000\mu\text{s}$).
  - Clip parameters: `timeline_start`, `source_in`, `source_out`.
  - Non-redundant duration and timeline_end computation.
- [x] Domain Invariants & Validation:
  - `source_in >= 0`, `source_out > source_in`, `timeline_start >= 0`.
  - Media duration boundary validation (`source_out <= media.duration`).
  - Track type compatibility (`VIDEO` tracks require video media).
  - Cross-project media leakage prevention (`media.project_id == project.id`).
- [x] Same-Track Overlap Policy:
  - Strict rejection of overlapping clips on the same track (`ErrClipOverlap` / HTTP 409).
  - Abutting clips allowed.
  - Multi-track layering permitted across different tracks.
- [x] Full REST API:
  - `GET /projects/:projectId/timeline` (Get complete timeline hierarchy or auto-initialize)
  - `POST /projects/:projectId/timeline` (Create timeline)
  - `POST /projects/:projectId/timeline/tracks` (Create track)
  - `PATCH /projects/:projectId/timeline/tracks/:trackId` (Update track)
  - `DELETE /projects/:projectId/timeline/tracks/:trackId` (Delete track)
  - `POST /projects/:projectId/timeline/tracks/:trackId/clips` (Create clip)
  - `PATCH /projects/:projectId/timeline/tracks/:trackId/clips/:clipId` (Update clip)
  - `DELETE /projects/:projectId/timeline/tracks/:trackId/clips/:clipId` (Delete clip)
- [x] Comprehensive Testing:
  - Go domain tests (timebase, invariants, overlap logic).
  - Go service tests (ownership validation, overlap rejection, cross-project protection).
  - Go handler tests (HTTP requests, status codes, error handling).
- [x] Frontend Types & API Client (`apps/web/src/types/timeline.ts`, `apps/web/src/api/timeline.ts`).

---

### Previous Deliverables
- **Hop 0 / 0.1:** Foundational Monorepo Setup, dependency boundaries, Go API, Python AI service, React frontend, `@ai-filmmaker/film-dsl`, `@ai-filmmaker/editor-core`.
- **Hop 1:** Project System, PostgreSQL migrations, project CRUD endpoints, frontend Project List and Project Shell.
- **Hop 2:** Media Ingestion, storage abstraction, FFmpeg proxy/thumbnail pipeline, background queue, path traversal hardening.

---

### Intentional Limitations & Deferred Features
1. **Visual NLE Editor / WebGPU Timeline:** Drag-and-drop timeline, trimming UI, waveform renderers, and WebGPU canvas playback are deferred to upcoming NLE sub-hops (Hop 3B+).
2. **Compound / Nested Clips:** Timeline sequences are single primary timeline per project.
3. **Authentication / Authorization:** Multi-tenant user auth (Clerk) remains deferred; media and timelines belong to projects without user-level access control.
4. **AI Generation / Vision Analysis:** Scene detection, shot tagging, Whisper speech-to-text, and embeddings remain deferred to AI orchestration hops.

---

### Next Sub-Hop: Hop 3B
- Editor Core / Timeline Engine integration
- WebCodecs video frame decoding
- Canvas / WebGPU timeline preview renderer
