# AI Filmmaker - Local Health Verification Script

Write-Host "Verifying AI Filmmaker Services Health..." -ForegroundColor Cyan

# 1. Web App
try {
    $webResp = Invoke-WebRequest -Uri "http://localhost:3000" -UseBasicParsing -TimeoutSec 3
    Write-Host "[OK] Web Frontend: UP (Status: $($webResp.StatusCode))" -ForegroundColor Green
} catch {
    Write-Host "[WARN] Web Frontend: NOT REACHABLE (Run 'npm run dev' in apps/web)" -ForegroundColor Yellow
}

# 2. Go API
try {
    $apiResp = Invoke-RestMethod -Uri "http://localhost:8080/health" -TimeoutSec 3
    Write-Host "[OK] Go API (/health): UP - Service: $($apiResp.service), Status: $($apiResp.status)" -ForegroundColor Green
} catch {
    Write-Host "[WARN] Go API: NOT REACHABLE (Run 'go run ./cmd/server' in apps/api)" -ForegroundColor Yellow
}

# 3. Python AI
try {
    $aiResp = Invoke-RestMethod -Uri "http://localhost:8000/health" -TimeoutSec 3
    Write-Host "[OK] Python AI (/health): UP - Service: $($aiResp.service), Status: $($aiResp.status)" -ForegroundColor Green
} catch {
    Write-Host "[WARN] Python AI: NOT REACHABLE (Run 'uvicorn main:app --port 8000' in apps/ai)" -ForegroundColor Yellow
}
