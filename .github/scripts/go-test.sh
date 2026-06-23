#!/usr/bin/env bash
set -euxo pipefail

MODE="${1:-test}"

# just for file path from stacks.
PROJECT_DIR="${PROJECT_DIR:-/home/runner/go/src/github.com/xgfone/go-toolkit}"

mkdir -p "$PROJECT_DIR"
rsync -a --exclude='.git' ./ "$PROJECT_DIR"/
cd "$PROJECT_DIR"

GO="$(command -v go)"

echo "GO=$GO"
"$GO" version
"$GO" env GOTOOLCHAIN GOROOT GOVERSION

"$GO" test -coverprofile=coverage.out -race ./...

if [ "$MODE" = "coverage" ]; then
  cp coverage.out "$GITHUB_WORKSPACE/coverage.out"
fi
