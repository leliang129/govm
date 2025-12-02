#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-dev}"
PLATFORMS=("linux/amd64" "linux/arm64" "linux/386")
DIST_DIR="dist"

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

for target in "${PLATFORMS[@]}"; do
  IFS="/" read -r GOOS GOARCH <<<"$target"
  BIN_NAME="govm-${VERSION}-${GOOS}-${GOARCH}"
  echo "Building $BIN_NAME"
  GOOS="$GOOS" GOARCH="$GOARCH" CGO_ENABLED=0 go build -o "$DIST_DIR/$BIN_NAME" ./cmd/govm
  if command -v shasum >/dev/null 2>&1; then
    (cd "$DIST_DIR" && shasum -a 256 "$BIN_NAME" > "$BIN_NAME.sha256")
  elif command -v sha256sum >/dev/null 2>&1; then
    (cd "$DIST_DIR" && sha256sum "$BIN_NAME" > "$BIN_NAME.sha256")
  else
    echo "Warning: sha256 tool not found; skipping checksum for $BIN_NAME"
  fi
	tar -C "$DIST_DIR" -czf "$DIST_DIR/$BIN_NAME.tar.gz" "$BIN_NAME" "$BIN_NAME.sha256" 2>/dev/null || true
	rm "$DIST_DIR/$BIN_NAME"
  rm "$DIST_DIR/$BIN_NAME.sha256"
  echo "Created $DIST_DIR/$BIN_NAME.tar.gz"
done

echo "Artifacts are available in $DIST_DIR"
