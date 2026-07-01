#!/bin/bash
set -euo pipefail

BINARY_NAME="retrokit"
CMD_PATH="./retrokit/cmd/retrokit"
ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"

mkdir -p "$ROOT_DIR/bin"
cd "$ROOT_DIR"
go build -o "$ROOT_DIR/bin/$BINARY_NAME" "$CMD_PATH"
echo "Binary ready at: $ROOT_DIR/bin/$BINARY_NAME"
