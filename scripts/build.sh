#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DIST="$ROOT/dist"
LDFLAGS='-s -w -H windowsgui'

mkdir -p "$DIST"

echo "→ sync dashboard assets"
go generate ./internal/dashboard/...

echo "→ Windows manifest (DPI)"
go run github.com/tc-hib/go-winres@v0.3.3 make \
	--arch amd64 \
	--out "$ROOT/cmd/letter/rsrc"

echo "→ letter.exe (Windows amd64, WebView2 GUI)"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
	go build -ldflags="$LDFLAGS" -o "$DIST/letter.exe" "$ROOT/cmd/letter"

echo ""
ls -lh "$DIST/letter.exe"
