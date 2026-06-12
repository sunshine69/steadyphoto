#!/bin/bash
# =====================================================
# SteadyPhoto Docker Build & Push Script
# Builds the Docker image and pushes it to Docker Hub
# =====================================================

set -e

DOCKER_IMAGE="steadyphoto"
TAG="${1:-latest}"  # Default tag is "latest", can override: ./build-and-push.sh v0.1.0

echo "=============================================="
echo "   SteadyPhoto Docker Build & Push"
echo "=============================================="
echo ""
echo "Image: ${DOCKER_IMAGE}:${TAG}"
echo ""

# Check if logged in to Docker Hub
if ! docker info > /dev/null 2>&1; then
    echo "ERROR: Docker is not running. Please start Docker Desktop or your Docker daemon."
    exit 1
fi

# Try to detect login status (works on most systems)
#docker pull hello-world > /dev/null 2>&1 || true
#if ! docker image inspect "${DOCKER_IMAGE}:${TAG}" > /dev/null 2>&1; then
#    # Image doesn't exist locally — check if we're logged in
#    if [ -z "$DOCKER_USERNAME" ]; then
#        echo "WARNING: DOCKER_USERNAME not set. Please set it or log in manually with 'docker login'."
#        echo ""
#        read -p "Press Enter to continue anyway (build will fail on push if not authenticated)..." || true
#    fi
#fi

# Build the image
echo "--- Step 1: Building Docker Image ---"
if [ "$TAG" = "latest" ]; then
    # For latest tag, don't include version in build args
    docker build \
        --no-cache \
        -t "${DOCKER_IMAGE}:${TAG}" \
        .
else
    # For versioned tags, pass the version for Angular app metadata
    VERSION=$(echo "$TAG" | sed 's/^v//')  # Remove leading "v" if present
    docker build \
        --no-cache \
        --build-arg APP_VERSION="${VERSION}" \
        -t "${DOCKER_IMAGE}:${TAG}" \
        .
fi

# Tag for Docker Hub (if username is set)
if [ -n "$DOCKER_USERNAME" ]; then
    # Also tag as <username>/<image>:<tag> so it can be pushed directly
    docker tag "${DOCKER_IMAGE}:${TAG}" "${DOCKER_USERNAME}/${DOCKER_IMAGE}:${TAG}"

    # If the image was built with "latest" tag, also create a versioned alias
    if [ "$TAG" = "latest" ]; then
        # Try to extract version from git or use commit hash as fallback
        GIT_TAG=$(git describe --tags --always 2>/dev/null || echo "$(date +%Y%m%d)-$(git rev-parse --short HEAD)")
        docker tag "${DOCKER_IMAGE}:latest" "${DOCKER_USERNAME}/${DOCKER_IMAGE}:${GIT_TAG}"
    else
        # For versioned tags, also create a "latest" alias
        docker tag "${DOCKER_IMAGE}:${TAG}" "${DOCKER_IMAGE}:latest"
        docker tag "${DOCKER_USERNAME}/${DOCKER_IMAGE}:${TAG}" "${DOCKER_USERNAME}/${DOCKER_IMAGE}:latest"
    fi
fi

# Push the image to Docker Hub (if username is set)
if [ -n "$DOCKER_USERNAME" ]; then
    echo ""
    echo "--- Step 2: Pushing to Docker Hub ---"

    # Push latest tag
    docker push "${DOCKER_USERNAME}/${DOCKER_IMAGE}:latest"

    # Push versioned tags if not already pushed as part of latest build
    if [ "$TAG" != "latest" ]; then
        docker push "${DOCKER_USERNAME}/${DOCKER_IMAGE}:${TAG}"
    fi

    echo ""
    echo "=============================================="
    echo "✅ Build & Push Complete!"
    echo "----------------------------------------------"
    echo "📦 Images available at:"
    echo "   https://hub.docker.com/r/${DOCKER_USERNAME}/${DOCKER_IMAGE}"
    echo "=============================================="
else
    echo ""
    echo "--- Step 2: Skipping push (no DOCKER_USERNAME set) ---"
    echo ""
    echo "To push, run with:"
    echo "  export DOCKER_USERNAME=your-dockerhub-username"
    echo "  ./build-and-push.sh [tag]"
fi
