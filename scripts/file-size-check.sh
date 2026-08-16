#!/bin/bash
# File size check: no file > 500 lines
# Block PRs that violate the modularity rule.

set -euo pipefail
cd "$(dirname "$0")/.."

MAX_LINES=500
FAIL=0

# Backend (Go)
while IFS= read -r file; do
    lines=$(wc -l < "$file")
    if [ "$lines" -gt "$MAX_LINES" ]; then
        echo "TOO BIG ($lines): $file"
        FAIL=1
    fi
done < <(find . -name "*.go" \
    -not -path "./vendor/*" \
    -not -name "*_test.go" \
    -not -name "*.pb.go" \
    -not -name "*_generated.go" \
    -not -name "*.bak*" \
    -not -path "*/node_modules/*")

# Frontend (TS/TSX/CSS)
while IFS= read -r file; do
    lines=$(wc -l < "$file")
    if [ "$lines" -gt "$MAX_LINES" ]; then
        echo "TOO BIG ($lines): $file"
        FAIL=1
    fi
done < <(find web/src -type f \( -name "*.ts" -o -name "*.tsx" -o -name "*.css" \) \
    -not -path "*/node_modules/*")

if [ "$FAIL" -eq 1 ]; then
    echo "FILE SIZE CHECK FAILED — split files into smaller modules"
    exit 1
fi

echo "FILE SIZE CHECK PASSED (max $MAX_LINES lines per file)"
