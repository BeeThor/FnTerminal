#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
export GOMODCACHE="${GOMODCACHE:-$PWD/.cache/go-mod}" GOCACHE="${GOCACHE:-$PWD/.cache/go-build}"
npm run check
npm test
python3 scripts/check-upstream.py
for script in packaging/cmd/* packaging/app/bin/nas-login scripts/*.sh; do bash -n "$script"; done
go mod verify
go vet ./...
go test -race -count=1 -timeout=90s ./...
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/fnterminal-check-amd64 ./cmd/fnterminal
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/fnterminal-check-arm64 ./cmd/fnterminal
