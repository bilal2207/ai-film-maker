from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(
    title="AI Filmmaker AI Service",
    description="AI and ML orchestration service for AI Filmmaker",
    version="0.1.0",
)


class HealthResponse(BaseModel):
    status: str
    service: str


@app.get("/health", response_model=HealthResponse)
async def health_check() -> HealthResponse:
    return HealthResponse(status="ok", service="ai")


if __name__ == "__main__":
    import uvicorn

    uvicorn.run("main:app", host="0.0.0.0", port=8000, reload=True)
