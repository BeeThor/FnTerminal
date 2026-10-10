#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version=1.2.3
case "$(uname -s)" in Darwin) os=darwin;; Linux) os=linux;; *) echo '仅支持 macOS/Linux 构建机' >&2; exit 1;; esac
case "$(uname -m)" in arm64|aarch64) arch=arm64;; x86_64|amd64) arch=amd64;; *) exit 1;; esac
mkdir -p .tools
curl --fail --location --proto '=https' --tlsv1.2 "https://static2.fnnas.com/fnpack/fnpack-$version-$os-$arch" -o .tools/fnpack.tmp
chmod 755 .tools/fnpack.tmp
mv .tools/fnpack.tmp .tools/fnpack
.tools/fnpack --help
# Official documentation publishes HTTPS downloads without a signed checksum.
# Record this download's SHA256 as provenance, not independent authenticity proof.
shasum -a 256 .tools/fnpack > .tools/fnpack.sha256
