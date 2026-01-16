Write-Host "🚀 Starting Klokku installation..." -ForegroundColor Cyan

# 1. Check if Docker is running
if (!(docker info 2>$null)) {
    Write-Host "❌ Error: Docker is not running. Please start Docker and try again." -ForegroundColor Red
    exit
}

# 2. Download docker-compose.yml
Write-Host "📥 Downloading docker-compose.yml..."
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/docker-compose.yml" -OutFile "docker-compose.yml"

# 3. Download .env.template and rename to .env (only if .env doesn't exist)
if (-not (Test-Path ".env")) {
    Write-Host "📝 Creating .env from template..."
    Invoke-WebRequest -Uri "https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/.env.template" -OutFile ".env"
} else {
    Write-Host "ℹ️  .env already exists, skipping download to protect your settings." -ForegroundColor Yellow
}

# 4. Start the containers
Write-Host "🐋 Starting Docker containers..."
docker compose up -d

if ($LASTEXITCODE -eq 0) {
    Write-Host "✅ Klokku is starting!" -ForegroundColor Green
    Write-Host "🔗 Access it at: http://localhost:8181"
} else {
    Write-Host "❌ Failed to start Docker containers." -ForegroundColor Red
}