#!/usr/bin/env bash
# Run all local release gates before marking an artifact ready for user testing.
set -euo pipefail
cd "$(dirname "$0")/.."
export GOMODCACHE="${GOMODCACHE:-$PWD/.cache/go-mod}" GOCACHE="${GOCACHE:-$PWD/.cache/go-build}"
mkdir -p .cache/release
./scripts/check.sh | tee .cache/release/check.log
npm run build
mkdir -p .tools
go test -c -o .tools/browser-tests ./cmd/fnterminal
npm run test:browser
linux_vm=not_run
if [[ $(uname -s) == Darwin && $(uname -m) == arm64 ]]; then
  ./scripts/test-linux-vm.sh
  linux_vm=passed
fi
./scripts/build-fpk.sh x86
./scripts/build-fpk.sh arm
python3 scripts/release-report.py "$linux_vm"
python3 scripts/package-source.py
