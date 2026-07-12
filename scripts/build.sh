#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
LDFLAGS='-s -w -H windowsgui'

mkdir -p "$DIST"

echo "→ sync dashboard assets"
go generate ./internal/dashboard/...

echo "→ letter.exe (Windows amd64, WebView2 GUI)"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
	go build -ldflags="$LDFLAGS" -o "$DIST/letter.exe" "$ROOT/cmd/letter"

echo ""
ls -lh "$DIST/letter.exe"
