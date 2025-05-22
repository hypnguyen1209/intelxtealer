#!/bin/bash
# Intel Stealer Docker Deployment Script

# Check for Docker and Docker Compose
if ! command -v docker &> /dev/null || ! command -v docker-compose &> /dev/null; then
    echo "Docker and/or Docker Compose are not installed. Please install them first."
    echo "Visit https://docs.docker.com/get-docker/ for installation instructions."
    exit 1
fi

# Create necessary directories if they don't exist
mkdir -p backend/data
mkdir -p nginx

# Build and start services
echo "Building and starting Docker services..."
docker-compose up -d --build

# Check if services are running
echo "Checking service status..."
docker-compose ps

echo ""
echo "Intel Stealer is now running!"
echo "Access the web interface at: http://localhost"
echo ""
echo "Useful commands:"
echo "- View logs: docker-compose logs -f"
echo "- Stop services: docker-compose down"
echo "- Restart services: docker-compose restart"
