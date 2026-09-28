# AI Filmmaker - Architecture Documentation

## System Overview

AI Filmmaker is an AI-native filmmaking platform designed around strict subsystem boundaries:

```text
React / TypeScript (apps/web)
        ↓ (REST API)
      Go API (apps/api)
        ↓
 PostgreSQL / Redis (infrastructure)
        ↓ (HTTP / gRPC)
 Python AI services (apps/ai)
```

---

## Core Layers & Responsibilities

### 1. Frontend (`apps/web`)
- **Technology:** React 18, TypeScript, Vite.
- **Role:** Presents user interfaces for project management, film storyboards, and editor views.
- **Boundaries:** Connects *exclusively* to the Go API (`VITE_API_URL`). Never calls Python AI services directly, and never connects directly to databases or caches.

### 2. Core API Backend (`apps/api`)
- **Technology:** Idiomatic Go standard library (`net/http`), `database/sql`, `github.com/lib/pq`, `github.com/google/uuid`.
- **Role:** System of record coordinator, REST API gateway, business logic orchestrator, and database manager.
- **Structure:**
  - `cmd/server/main.go`: Application bootstrap, database connection pooling, auto-migration trigger, and server lifecycle.
  - `internal/config/`: Environment configuration loading.
  - `internal/domain/`: Domain entities and request validation logic (`Project`, `CreateProjectInput`, `UpdateProjectInput`).
  - `internal/repository/`: Abstract persistence interfaces (`ProjectRepository`) and PostgreSQL implementations (`PostgresProjectRepository`).
  - `internal/service/`: Business rules, validation, timestamp orchestration, and UUID generation (`ProjectService`).
  - `internal/handlers/`: HTTP request dispatching, JSON decoding, status code management, and structured error responses.
  - `internal/database/migrations/`: Embeddable transactional SQL migration runner.

### 3. Python AI Service (`apps/ai`)
- **Technology:** Python 3.10+, FastAPI, `uv`.
- **Role:** AI/ML workload orchestration, model inference, and Film DSL generation.
- **Boundaries:** Only reachable by internal Go backend calls (via HTTP/gRPC).

---

## Hop 1 Architectural Decisions

1. **Layered Separation in Go:** The Go API strictly separates repository, service, and HTTP layers.
2. **Standard Library Routing:** Leveraged Go's native `http.ServeMux` for URL dispatching with lightweight CORS support.
3. **No Direct Frontend-to-AI Connectivity:** Preserved architectural constraint that all frontend requests must pass through Go API.
4. **Temporary Auth Assumption:** Authentication is deferred to future hops. Currently, projects are managed globally for development.
