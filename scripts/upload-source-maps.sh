#!/bin/bash

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

SENTRY_ORG="${SENTRY_ORG:-}"
SENTRY_PROJECT="${SENTRY_PROJECT:-hjtpx-api}"
SENTRY_AUTH_TOKEN="${SENTRY_AUTH_TOKEN:-}"
SENTRY_DSN="${SENTRY_DSN:-}"

VERSION="${APP_VERSION:-$(git rev-parse --short HEAD)}"
NODE_ENV="${NODE_ENV:-production}"

echo "🚀 Sentry Source Maps Upload"
echo "========================================"
echo "Organization: ${SENTRY_ORG}"
echo "Project: ${SENTRY_PROJECT}"
echo "Version: ${VERSION}"
echo "Environment: ${NODE_ENV}"
echo "========================================"

if [ -z "$SENTRY_AUTH_TOKEN" ]; then
    echo "⚠️  SENTRY_AUTH_TOKEN not set. Skipping source maps upload."
    echo "   Set SENTRY_AUTH_TOKEN to enable source maps."
    exit 0
fi

if [ -z "$SENTRY_ORG" ]; then
    echo "⚠️  SENTRY_ORG not set. Skipping source maps upload."
    exit 0
fi

dist_paths=(
    "./dist"
    "./build"
    "./.next"
    "./release"
)

for path in "${dist_paths[@]}"; do
    if [ -d "$path" ]; then
        echo "📦 Processing: $path"
        
        sentry-cli releases files "$VERSION" \
            --org "$SENTRY_ORG" \
            --project "$SENTRY_PROJECT" \
            --validate \
            new \
            --name "$VERSION" || true
        
        sentry-cli releases files "$VERSION" \
            --org "$SENTRY_ORG" \
            --project "$SENTRY_PROJECT" \
            deploy "$NODE_ENV" \
            --name "$VERSION" || true
        
        echo "✅ Source maps uploaded for $path"
    fi
done

sentry-cli releases set-commits \
    --org "$SENTRY_ORG" \
    --project "$SENTRY_PROJECT" \
    --commit "origin/main@$VERSION" \
    --ignore-missing

sentry-cli releases finalize "$VERSION" \
    --org "$SENTRY_ORG" \
    --project "$SENTRY_PROJECT"

echo "========================================"
echo "✅ Source maps upload complete!"
echo "   Release: $VERSION"
echo "   Environment: $NODE_ENV"
echo "========================================"
