# AI Filmmaker

> **AI-native filmmaking platform:**
> Idea → Script → Production → Footage → Edit → VFX → Color → Sound → Distribution.

The eventual product will contain its own non-linear editor (NLE), Film Intelligence layer, AI orchestration, Film DSL, media processing pipeline, and voice-first filmmaking interface.

---

## Architecture Overview

```text
React / TypeScript (apps/web)
        ↓ (REST / WebSocket)
      Go API (apps/api)
        ↓
 PostgreSQL / Redis (infrastructure)
        ↓ (HTTP / gRPC)
 Python AI services (apps/ai)
```

- **`apps/web`**: React, TypeScript, and Vite frontend providing the user and playback interface.
- **`apps/api`**: Go core backend orchestrating business logic, relational storage, and API routing.
- **`apps/ai`**: Python FastAPI service orchestrating AI/ML generation models and tasks.
- **`packages/film-dsl`**: Typed domain-specific language bridging AI generation and editing commands.
- **`packages/editor-core`**: Core timeline, state management, and media playback foundation.
- **`infrastructure/`**: Local Docker Compose definitions (PostgreSQL 16, Redis 7) and database migrations.

---

## Repository Structure

```text
ai-filmmaker/
├── apps/
│   ├── web/                     # React + TypeScript + Vite frontend
│   ├── api/                     # Go REST API service
│   └── ai/                      # Python FastAPI AI service
├── packages/
│   ├── film-dsl/                # Film DSL contract & schemas
│   └── editor-core/             # Editor timeline foundation
├── infrastructure/
│   ├── docker-compose.yml       # Local PostgreSQL and Redis
│   └── migrations/              # Database migration SQL files
├── docs/
│   └── DEVELOPMENT-RULES.md     # Core engineering principles
├── scripts/
│   └── check_health.ps1         # System health verification script
├── .github/
│   └── workflows/ci.yml         # GitHub Actions CI workflow
├── AGENTS.md                    # Agent architecture guidelines & constraints
├── PROJECT-STATUS.md            # Hop progression & project status
├── .env.example                 # Environment configuration template
└── README.md                    # Project documentation
```

---

## Prerequisites

- **Node.js** 20+ & **npm**
- **Go** 1.22+
- **Python** 3.10+ (uv or pip)
- **Docker** & **Docker Compose**

---

## Installation & Setup

1. **Clone the repository:**
   ```bash
   git clone <repo-url>
   cd ai-film-maker
   ```

2. **Configure Environment Variables:**
   ```bash
   cp .env.example .env
   ```

3. **Install Frontend Dependencies:**
   ```bash
   cd apps/web
   npm install
   ```

4. **Install Python AI Dependencies:**
   ```bash
   cd apps/ai
   pip install -e ".[dev]"
   # or with uv: uv pip install -e ".[dev]"
   ```

5. **Download Go Modules:**
   ```bash
   cd apps/api
   go mod tidy
   ```

---

## Local Development

### 1. Start Infrastructure (PostgreSQL & Redis)
```bash
docker compose -f infrastructure/docker-compose.yml up -d
```

### 2. Start Services

- **Web Frontend** (Port 3000):
  ```bash
  cd apps/web
  npm run dev
  ```

- **Go API Service** (Port 8080):
  ```bash
  cd apps/api
  go run ./cmd/server
  ```

- **Python AI Service** (Port 8000):
  ```bash
  cd apps/ai
  uvicorn main:app --port 8000 --reload
  ```

---

## Health Check URLs

| Service | Protocol | Endpoint | Expected Response |
| :--- | :--- | :--- | :--- |
| **Web Frontend** | HTTP | `http://localhost:3000` | Rendered UI ("AI Filmmaker - Foundation ready.") |
| **Go API** | REST | `http://localhost:8080/health` | `{"status": "ok", "service": "api"}` |
| **Python AI** | REST | `http://localhost:8000/health` | `{"status": "ok", "service": "ai"}` |
| **PostgreSQL** | TCP | `localhost:5432` | `pg_isready` ok |
| **Redis** | TCP | `localhost:6379` | `PONG` |

---

## Running Tests

- **Frontend Tests:**
  ```bash
  cd apps/web
  npm run test
  ```

- **Go API Tests:**
  ```bash
  cd apps/api
  go test -v ./...
  ```

- **Python AI Tests:**
  ```bash
  cd apps/ai
  pytest -v
  ```

---

## Running Linters & Formatters

- **Frontend (ESLint & TypeScript Check):**
  ```bash
  cd apps/web
  npm run lint
  npx tsc --noEmit
  ```

- **Go (gofmt & go vet):**
  ```bash
  cd apps/api
  gofmt -s -w .
  go vet ./...
  ```

- **Python (Ruff):**
  ```bash
  cd apps/ai
  ruff check .
  ruff format --check .
  ```
