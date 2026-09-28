# AI Filmmaker - Python AI Service

This service orchestrates AI/ML workloads and model inferences for the AI Filmmaker platform.

---

## Development Setup

We use [`uv`](https://github.com/astral-sh/uv) for fast, reproducible Python environment and dependency management.

### Prerequisites

- Python 3.10+
- `uv` installed (`pip install uv` or via standalone binary / package manager)

### Installation

From the `apps/ai` directory:

```bash
uv sync
```

### Running Tests

```bash
uv run pytest
```

### Linting & Formatting

```bash
uv run ruff check .
uv run ruff format --check .
```

To auto-format:

```bash
uv run ruff format .
```

### Running the Service

```bash
uv run uvicorn main:app --port 8000 --reload
```

Health endpoint: `http://localhost:8000/health`
