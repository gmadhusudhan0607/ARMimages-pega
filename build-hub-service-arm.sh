#!/usr/bin/env bash
#
# Build ARM64 image for gen-ai-hub-service (includes hub-service + gateway-ops)
#
# Prerequisites:
#   - Docker with BuildKit enabled (Docker 19.03+)
#   - docker buildx (comes with Docker Desktop; on Linux: docker buildx create --use)
#   - QEMU registered (only if building on x86):
#       docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
#
# Usage:
#   ./build-hub-service-arm.sh                          # Build and load locally
#   ./build-hub-service-arm.sh --push                   # Build and push to registry
#   ./build-hub-service-arm.sh --tag v1.0.0             # Custom image tag
#   ./build-hub-service-arm.sh --registry <ECR_URL>     # Custom registry
#
set -euo pipefail

# ========================== CONFIGURATION ===========================
TARGET_PLATFORM="linux/arm64"

# AWS ECR registry
REGISTRY="${REGISTRY:-349534214040.dkr.ecr.us-east-1.amazonaws.com}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REPO_NAME="genai-hub-service-arm64"

# Git credentials for private Go modules
BITBUCKET_USR="${BITBUCKET_USR:-}"
BITBUCKET_PSW="${BITBUCKET_PSW:-}"
GITHUB_USR="${GITHUB_USR:-}"
GITHUB_PSW="${GITHUB_PSW:-}"

# Source code root
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HUB_SERVICE_ROOT="${SCRIPT_DIR}/gen-ai-hub-service-main/gen-ai-hub-service-main"

# ========================== PARSE ARGS ==============================
OUTPUT_MODE="--load"

while [[ $# -gt 0 ]]; do
    case "$1" in
        --push)     OUTPUT_MODE="--push"; shift ;;
        --load)     OUTPUT_MODE="--load"; shift ;;
        --tag)      IMAGE_TAG="$2"; shift 2 ;;
        --registry) REGISTRY="$2"; shift 2 ;;
        -h|--help)
            echo "Usage: $0 [--push|--load] [--tag TAG] [--registry REGISTRY]"
            exit 0 ;;
        *) echo "Unknown option: $1"; exit 1 ;;
    esac
done

# ========================== FUNCTIONS ===============================
ensure_buildx() {
    echo "==> Ensuring Docker Buildx builder exists..."
    if ! docker buildx inspect arm-builder &>/dev/null; then
        docker buildx create --name arm-builder --platform linux/arm64,linux/amd64 --use
        docker buildx inspect --bootstrap arm-builder
    else
        docker buildx use arm-builder
    fi
}

ecr_login() {
    echo "==> Logging into ECR: ${REGISTRY}..."
    aws ecr get-login-password --region us-east-1 | docker login --username AWS --password-stdin "${REGISTRY}"
}

build_image() {
    local image_name="$1"
    local dockerfile_path="$2"
    local context_dir="$3"
    local full_image="${REGISTRY}/${image_name}:${IMAGE_TAG}"

    echo ""
    echo "================================================================"
    echo "  Building ARM64 image: ${full_image}"
    echo "  Dockerfile: ${dockerfile_path}"
    echo "================================================================"

    docker buildx build \
        --platform "${TARGET_PLATFORM}" \
        --file "${dockerfile_path}" \
        --build-arg BITBUCKET_USR="${BITBUCKET_USR}" \
        --build-arg BITBUCKET_PSW="${BITBUCKET_PSW}" \
        --build-arg GITHUB_USR="${GITHUB_USR}" \
        --build-arg GITHUB_PSW="${GITHUB_PSW}" \
        --tag "${full_image}" \
        ${OUTPUT_MODE} \
        "${context_dir}"

    echo "==> Successfully built: ${full_image}"
}

# ========================== BUILD ==================================
ensure_buildx
ecr_login

# Prepare Docker build context - Dockerfiles expect COPY libs/...
if [[ ! -d "${HUB_SERVICE_ROOT}/libs" ]]; then
    echo "==> Creating libs/ symlink for Docker build context..."
    ln -sfn "${HUB_SERVICE_ROOT}" "${HUB_SERVICE_ROOT}/libs"
fi

echo ""
echo "############################################################"
echo "#  Building gen-ai-hub-service ARM64 images"
echo "############################################################"

# Build genai-hub-service ARM64 image (main LLM gateway - port 8080)
# Both service and ops binaries are built into the same repo: genai-hub-service-arm64
build_image \
    "${REPO_NAME}" \
    "${HUB_SERVICE_ROOT}/distribution/genai-hub-service-docker/src/main/docker/Dockerfile" \
    "${HUB_SERVICE_ROOT}"

echo ""
echo "============================================================"
echo "  gen-ai-hub-service ARM64 images built successfully!"
echo ""
echo "  Image:"
echo "    - ${REGISTRY}/${REPO_NAME}:${IMAGE_TAG}"
echo ""
if [[ "${OUTPUT_MODE}" == "--push" ]]; then
    echo "  Images pushed to ${REGISTRY}"
else
    echo "  Images loaded into local Docker."
    echo "  To push: $0 --push --registry ${REGISTRY}"
fi
echo "============================================================"
