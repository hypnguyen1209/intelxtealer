# Intel Stealer Docker Deployment PowerShell Script

# Check for Docker
if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    Write-Host "Docker is not installed or not in your PATH. Please install Docker Desktop first."
    Write-Host "Visit https://docs.docker.com/docker-for-windows/install/ for installation instructions."
    exit 1
}

# Create necessary directories if they don't exist
if (-not (Test-Path -Path ".\backend\data")) {
    New-Item -ItemType Directory -Path ".\backend\data" -Force
}

if (-not (Test-Path -Path ".\nginx")) {
    New-Item -ItemType Directory -Path ".\nginx" -Force
}

# Build and start services
Write-Host "Building and starting Docker services..."
docker-compose up -d --build

# Check if services are running
Write-Host "Checking service status..."
docker-compose ps

Write-Host ""
Write-Host "Intel Stealer is now running!"
Write-Host "Access the web interface at: http://localhost"
Write-Host ""
Write-Host "Useful commands:"
Write-Host "- View logs: docker-compose logs -f"
Write-Host "- Stop services: docker-compose down"
Write-Host "- Restart services: docker-compose restart"
