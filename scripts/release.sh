#!/bin/bash
# release.sh — build, tag, and verify a StackWatch release.
# Used by CI on main merges (see .github/workflows/release.yml).

set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 v1.2.3"
    exit 1
fi

echo "[release] Building StackWatch $VERSION"

# Verify clean state
if [[ -n "$(git status --porcelain)" ]]; then
    echo "ERROR: working tree not clean — commit or stash before release"
    exit 1
fi
if [[ "$(git branch --show-current)" != "main" ]]; then
    echo "ERROR: not on main — release only from main"
    exit 1
fi

echo "[release] Running tests..."
go test ./... -count=1 -race -timeout=120s

echo "[release] Running vet+lint+fmt..."
gofmt -l . | (! grep .) || (echo "gofmt violations found"; exit 1)
go vet ./...
golangci-lint run --timeout=5m ./...

echo "[release] Cross-compiling all binaries..."
mkdir -p bin/release/$VERSION
for svc in api-gateway agent web-terminal truenas-connector; do
  for goos in linux windows darwin; do
    for goarch in amd64 arm64; do
      out="bin/release/$VERSION/stackwatch-${svc}-${goos}-${goarch}"
      if [[ "$goos" == "windows" ]]; then out="$out.exe"; fi
      CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" go build -o "$out" "./cmd/${svc}"
    done
  done
done

echo "[release] Writing GitHub release notes..."
cat > bin/release/$VERSION/RELEASE-NOTES.md << EOF
# StackWatch $VERSION

Built from commit: $(git rev-parse HEAD)
Date: $(date -u +%Y-%m-%dT%H:%M:%SZ)

See CHANGELOG.md for breaking changes.

## Quick check
\`\`\`bash
./stackwatch-api-gateway-linux-amd64 --health
\`\`\`
EOF

echo "[release] Tagging..."
git tag -a "$VERSION" -m "StackWatch $VERSION"
git push origin "$VERSION"

echo "[release] Done. Artifacts in bin/release/$VERSION/"
sha256sum bin/release/$VERSION/* > bin/release/$VERSION/SHA256SUMS.txt
echo "SHA256SUMS.txt written"
