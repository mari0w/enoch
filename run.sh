#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

if ! command -v go >/dev/null 2>&1; then
  echo "go not found in PATH"
  exit 1
fi

if [ ! -f ".env" ]; then
  echo ".env not found. Copy .env.example to .env and fill TELEGRAM_BOT_TOKEN"
  exit 1
fi

if ! command -v codex >/dev/null 2>&1; then
  echo "codex CLI not found in PATH"
  exit 1
fi

exec go run ./cmd/enoch
