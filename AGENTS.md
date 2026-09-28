# AI Filmmaker - Agent Guidelines & Architecture Boundaries

This document defines the architectural boundaries and operational rules for AI agents and developers working on the **AI Filmmaker** codebase.

---

## High-Level Architecture

```text
React / TypeScript (apps/web)
        ↓ (REST / WebSocket)
      Go API (apps/api)
        ↓
 PostgreSQL / Redis (infrastructure)
        ↓ (HTTP / gRPC)
 Python AI services (apps/ai)
```

### Future Evolution Pipeline

```text
AI Orchestration (apps/ai)
        ↓
Film DSL (packages/film-dsl)
        ↓
Editor Core (packages/editor-core)
        ↓
Timeline / NLE (WebGPU / WebCodecs)
```

---

## Critical Agent Rules

1. **Do not bypass architecture boundaries:**
   - Frontend must never call Python AI services directly; all requests flow through the Go API gateway/core backend.
   - Frontend must never interact directly with databases or caches.

2. **Do not allow AI to directly manipulate UI state:**
   - AI outputs structured decisions via the Film DSL contracts.
   - The Editor Core and UI consume Film DSL declarations deterministically.

3. **Do not introduce microservices unnecessarily:**
   - Keep the system bounded to `apps/web`, `apps/api`, and `apps/ai`.

4. **Do not introduce Kubernetes or heavy orchestrators:**
   - Maintain a simple Docker / Docker Compose setup until scale requires complexity.

5. **Do not introduce Kafka / RabbitMQ unless explicitly approved:**
   - Redis and PostgreSQL handle caching, pub/sub, and persistent state for initial hops.

6. **Do not replace PostgreSQL with another database:**
   - PostgreSQL is the system of record.
   - Binary media belongs in object storage (S3), not in PostgreSQL.

7. **Do not replace the Go backend without approval:**
   - Go handles core business logic, synchronization, and external REST APIs.

8. **Do not replace the Python AI service without approval:**
   - Python handles AI/ML workloads, model orchestration, and generation tasks.

9. **Do not add frameworks merely for convenience:**
   - Keep the dependency footprint lean and idiomatic.

10. **Keep modules small and testable:**
    - Every new feature, endpoint, or package function must include unit tests.

11. **Ask and stop when an architectural decision is ambiguous:**
    - Never guess or silently expand system boundaries without explicit approval.
