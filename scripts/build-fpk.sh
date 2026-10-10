#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
arch="${1:-x86}"
case "$arch" in x86) goarch=amd64;; arm) goarch=arm64;; *) echo 'usage: build-fpk.sh x86|arm' >&2; exit 2;; esac
fnpack_bin="${FNPACK:-$PWD/.tools/fnpack}"
if [[ ! -x "$fnpack_bin" ]]; then fnpack_bin=$(command -v fnpack) || { echo '先运行 scripts/download-fnpack.sh' >&2; exit 1; }; fi
export GOMODCACHE="${GOMODCACHE:-$PWD/.cache/go-mod}" GOCACHE="${GOCACHE:-$PWD/.cache/go-build}"
npm ci --ignore-scripts --no-audit --no-fund
npm run build
stage="$PWD/.build/$arch"
rm -rf "$stage"
mkdir -p "$stage" dist
cp -R packaging/. "$stage/"
mkdir -p "$stage/wizard" "$stage/app/web" "$stage/app/licenses"
cp -R .build/web/. "$stage/app/web/"
GOOS=linux GOARCH="$goarch" CGO_ENABLED=0 go build -trimpath -ldflags='-s -w -buildid=' -o "$stage/app/bin/fnterminal" ./cmd/fnterminal
python3 scripts/prepare-package.py "$stage" "$arch"
cp "$GOMODCACHE"/github.com/creack/pty@v1.1.24/LICENSE "$stage/app/licenses/creack-pty.txt"
cp "$GOMODCACHE"/github.com/gorilla/websocket@v1.5.3/LICENSE "$stage/app/licenses/gorilla-websocket.txt"
cp node_modules/@xterm/xterm/LICENSE "$stage/app/licenses/xterm.txt"
cp node_modules/@xterm/addon-fit/LICENSE "$stage/app/licenses/addon-fit.txt"
cp node_modules/@xterm/addon-search/LICENSE "$stage/app/licenses/addon-search.txt"
cp third_party/uniterm/LICENSE "$stage/app/licenses/uniterm-Apache-2.0.txt"
cp third_party/uniterm/THIRD_PARTY_NOTICES.md "$stage/app/licenses/uniterm-THIRD-PARTY-NOTICES.md"
cp NOTICE "$stage/app/licenses/FnTerminal-NOTICE.txt"
cp node_modules/@fontsource-variable/jetbrains-mono/LICENSE "$stage/app/licenses/jetbrains-mono-OFL.txt"
node scripts/collect-licenses.mjs "$stage/app/licenses/npm"
chmod 755 "$stage"/cmd/* "$stage"/app/bin/*
version=$(sed -n 's/^version=//p' "$stage/manifest")
# fnpack writes <appname>.fpk in its working directory; keep architectures apart.
(cd "$stage" && "$fnpack_bin" build --directory "$stage")
cp "$stage/fnterminal.fpk" "dist/FnTerminal-$version-$arch.fpk"
python3 scripts/validate-fpk.py "dist/FnTerminal-$version-$arch.fpk" "$arch"
(cd dist && shasum -a 256 ./FnTerminal-"$version"-*.fpk > SHA256SUMS)
