#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"

VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")

echo "==> Building frontend..."
cd frontend
npm run build
cd ..

echo "==> Building backend (version: $VERSION)..."
cd backend
CGO_ENABLED=0 go build -ldflags "-X loop/internal/version.Version=$VERSION" -o ../loop .
cd ..

echo "==> Done! Binary: ./loop (version: $VERSION)"
