#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "=== Formatting ==="
go fmt ./...

echo "=== Vetting ==="
go vet ./...

echo "=== Testing ==="
go test ./...
echo "=== All quality checks passed ==="
