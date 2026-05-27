#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"

echo "==> Building frontend..."
cd frontend
npm run build
cd ..

echo "==> Building backend..."
cd backend
CGO_ENABLED=0 go build -o ../loop .
cd ..

echo "==> Done! Binary: ./loop"
