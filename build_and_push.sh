#!/bin/bash

# Exit on error
set -e

# Configuration
GHCR_USER="rocjay1"
APP_NAME="balance-tracker"
TAG="latest"
PLATFORM="linux/arm64"

echo "Building images for $PLATFORM..."

# Build and Push Backend
echo "Building and Pushing Backend..."
docker buildx build --platform $PLATFORM \
  -t "ghcr.io/$GHCR_USER/${APP_NAME}-backend:$TAG" \
  ./backend --push

# Build and Push Frontend
echo "Building and Pushing Frontend..."
docker buildx build --platform $PLATFORM \
  -t "ghcr.io/$GHCR_USER/${APP_NAME}-frontend:$TAG" \
  ./frontend --push

echo "Build and Push complete!"
echo ""
echo "Deployment is now managed via the infrastructure repository:"
echo "  /Users/roccodavino/Source/rocjay1-infrastucture/balance-tracker/deploy.sh"
echo ""
echo "Watchtower will also automatically pick up these changes on the Pi within 5 minutes."
