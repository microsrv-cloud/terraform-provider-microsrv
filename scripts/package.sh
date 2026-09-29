#!/usr/bin/env bash
# Build registry-format .zip packages (one per platform) for OCI publishing.
# Usage: scripts/package.sh <version>
set -euo pipefail

VERSION="${1:?usage: scripts/package.sh <version>}"
DIST="dist"
BINARY="terraform-provider-microsrv"
platforms=(linux_amd64 linux_arm64 darwin_amd64 darwin_arm64 windows_amd64)

rm -rf "$DIST"
for p in "${platforms[@]}"; do
  os="${p%_*}"
  arch="${p#*_}"
  ext=""
  if [ "$os" = windows ]; then ext=".exe"; fi
  mkdir -p "$DIST/pkg_$p"
  GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 \
    go build -ldflags="-X main.version=$VERSION" -o "$DIST/pkg_$p/$BINARY$ext" .
  (cd "$DIST/pkg_$p" && zip -q "../${BINARY}_${VERSION}_${p}.zip" "$BINARY$ext")
done
rm -rf "$DIST"/pkg_*
ls -1 "$DIST"
