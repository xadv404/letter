#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
LDFLAGS="-s -w"

mkdir -p "$DIST"

echo "→ letter (Linux GUI)"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="$LDFLAGS" -o "$DIST/letter" "$ROOT/cmd/letter"

if command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
	echo "→ letter.exe (Windows GUI)"
	CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
		go build -ldflags="$LDFLAGS" -o "$DIST/letter.exe" "$ROOT/cmd/letter"
else
	echo "→ letter.exe skipped (install mingw-w64 for Windows cross-compile)"
fi

echo ""
echo "Built:"
ls -lh "$DIST"/letter* 2>/dev/null || ls -lh "$DIST"/
