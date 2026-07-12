#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
LDFLAGS="-s -w"

mkdir -p "$DIST"

echo "→ letter (Linux amd64)"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$DIST/letter" "$ROOT/cmd/letter"

echo "→ letter.exe (Windows amd64)"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$DIST/letter.exe" "$ROOT/cmd/letter"

echo "→ letter-gui (Linux amd64, requires CGO)"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$DIST/letter-gui" "$ROOT/cmd/letter-gui"

echo ""
echo "Built:"
ls -lh "$DIST"/letter "$DIST"/letter.exe "$DIST"/letter-gui 2>/dev/null || ls -lh "$DIST"/
