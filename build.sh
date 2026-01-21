#!/bin/bash

# Build script for CodeSentry
# This script builds both frontend and backend together

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=== Building CodeSentry ==="

# Step 1: Build frontend
echo ""
echo ">>> Building frontend..."
cd frontend
npm ci
npm run build
cd ..

# Step 2: Copy frontend to backend static folder
echo ""
echo ">>> Copying frontend to backend/cmd/server/static/..."
rm -rf backend/cmd/server/static
cp -r frontend/dist backend/cmd/server/static

# Step 3: Build backend
echo ""
echo ">>> Building backend..."
cd backend
go build -o ../codesentry ./cmd/server/
cd ..

echo ""
echo "=== Build complete! ==="
echo ""
echo "Run with: ./codesentry"
echo "Default URL: http://localhost:8080"
echo "Default login: admin / admin"
