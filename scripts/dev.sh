#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
echo "→ sync dashboard"
go generate ./internal/dashboard/...
echo "→ run Letter Recon"
go run ./cmd/letter
