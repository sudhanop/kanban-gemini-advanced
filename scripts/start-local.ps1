# FlowBoard Local Development Startup Script (Windows PowerShell)
# Run from the project root: .\scripts\start-local.ps1

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot

Write-Host ""
Write-Host "  ⚡ FlowBoard — Local Development" -ForegroundColor Cyan
Write-Host "  ─────────────────────────────────────" -ForegroundColor DarkGray
Write-Host ""

# Check .env exists
if (-not (Test-Path "$Root\.env")) {
    Write-Host "  [ERROR] .env file not found in project root." -ForegroundColor Red
    Write-Host "          Copy .env.example to .env and fill in your values." -ForegroundColor Yellow
    exit 1
}

# Kill any existing processes on the ports
Write-Host "  Checking ports..." -ForegroundColor DarkGray
@(8080, 3000) | ForEach-Object {
    $port = $_
    $procs = netstat -ano | Select-String ":$port " | ForEach-Object {
        ($_ -split '\s+')[-1]
    } | Select-Object -Unique
    foreach ($pid in $procs) {
        if ($pid -match '^\d+$' -and $pid -ne 0) {
            try { Stop-Process -Id $pid -Force -ErrorAction SilentlyContinue } catch {}
        }
    }
}

Write-Host "  Starting backend (Go + SQLite)..." -ForegroundColor Green
$backend = Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$Root\backend'; go run main.go" -PassThru

Start-Sleep -Seconds 3

Write-Host "  Starting frontend (Next.js)..." -ForegroundColor Green
$frontend = Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$Root\frontend'; npm run dev" -PassThru

Write-Host ""
Write-Host "  ✓ Backend:  http://localhost:8080" -ForegroundColor Cyan
Write-Host "  ✓ Frontend: http://localhost:3000" -ForegroundColor Cyan
Write-Host "  ✓ Data:     $Root\backend\db.json" -ForegroundColor DarkGray
Write-Host ""
Write-Host "  Press Ctrl+C or close this window to stop." -ForegroundColor DarkGray
Write-Host ""

Wait-Process -Id $backend.Id, $frontend.Id
