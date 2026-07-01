#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

go clean
find bin/ -mindepth 1 ! -name '.gitkeep' -delete
echo "Done: cleaned build artifacts"
