#!/bin/bash

# Exit on error
set -e

# Configuration
APP_NAME="balance-tracker"
TAG="latest"
PLATFORM="linux/arm64"

echo "Building images for $PLATFORM..."

# Build Backend
echo "Building Backend..."
docker buildx build --platform $PLATFORM -t "${APP_NAME}-backend:$TAG" ./backend --load

# Build Frontend
echo "Building Frontend..."
docker buildx build --platform $PLATFORM -t "${APP_NAME}-frontend:$TAG" ./frontend --load

# Save images to a tarball
echo "Saving images to ${APP_NAME}-images.tar..."
docker save "${APP_NAME}-backend:$TAG" "${APP_NAME}-frontend:$TAG" > "${APP_NAME}-images.tar"

echo "Build complete! ${APP_NAME}-images.tar is ready."
echo ""
# Default host
PI_HOST="raspberrypi.local"

echo "Transferring files to ${PI_HOST}..."
scp balance-tracker-images.tar docker-compose.yml .env backend/config.yaml "${PI_HOST}:~/"

echo "Deploying on ${PI_HOST}..."
base64_cmd="
    docker load -i balance-tracker-images.tar
    docker compose up -d
"
ssh "${PI_HOST}" "${base64_cmd}"

echo "App deployed on ${PI_HOST}!"
