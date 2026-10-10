#!/bin/bash
set -euo pipefail
cd "$(dirname "$0")/.."
[[ $(uname -s) == Darwin && $(uname -m) == arm64 ]] || { echo '该测试机脚本仅用于 Apple Silicon macOS；Linux可直接执行Go测试。'; exit 1; }
export GOMODCACHE="${GOMODCACHE:-$PWD/.cache/go-mod}" GOCACHE="${GOCACHE:-$PWD/.cache/go-build}"
python3 scripts/linux/prepare-vm.py
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go test -c -o .test-linux/rootfs/fttests ./cmd/fnterminal
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o .test-linux/rootfs/production-service ./cmd/fnterminal
python3 scripts/linux/make-initramfs.py
swiftc scripts/linux/VM.swift -o .test-linux/vm -module-cache-path .test-linux/swift-cache
codesign --force --sign - --entitlements scripts/linux/VM.entitlements .test-linux/vm
.test-linux/vm .test-linux/kernel .test-linux/initramfs.cpio.gz > .test-linux/linux.log 2>&1
rg 'LINUX_TEST_RESULT=0' .test-linux/linux.log
mkdir -p test-results
cp .test-linux/linux.log test-results/linux.log
