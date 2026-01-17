#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "🚀 Starting Klokku installation..."

# 1. Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running. Please start Docker and try again."
    exit 1
fi

# 2. Download docker-compose.yml
echo "📥 Downloading docker-compose.yml..."
curl -sSLO https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/docker-compose.yml

# 3. Download init.sql for database setup
echo "📂 Preparing database initialization script..."
mkdir -p db
curl -sSL -o db/init.sql https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/db/init.sql

# 4. Download .env.template and rename to .env
if [ ! -f .env ]; then
    echo "📝 Creating .env from template..."
    curl -sSL -o .env https://raw.githubusercontent.com/klokku/klokku/refs/heads/main/.env.template
else
    echo "ℹ️  .env already exists, skipping download."
fi

# 5. Start the containers
echo "🐋 Starting Docker containers..."
if docker compose up -d; then
    echo "✅ Klokku is starting!"
    echo "🔗 Access it at: http://localhost:8181"
else
    echo "❌ Failed to start Docker containers."
    exit 1
fi