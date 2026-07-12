#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
LDFLAGS='-s -w -H windowsgui'

mkdir -p "$DIST"

if ! command -v x86_64-w64-mingw32-gcc >/dev/null 2>&1; then
	echo "error: install mingw — apt install gcc-mingw-w64-x86-64" >&2
	exit 1
fi

echo "→ letter.exe (Windows amd64 GUI)"
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc \
	go build -ldflags="$LDFLAGS" -o "$DIST/letter.exe" "$ROOT/cmd/letter"

echo ""
ls -lh "$DIST/letter.exe"
echo "Upload: gh release create vX.Y.Z dist/letter.exe --generate-notes"
