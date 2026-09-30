#!/usr/bin/env bash
# Сборка выпуска: воспроизводимые бинарники под Linux и Windows + контрольные суммы.
set -euo pipefail
cd "$(dirname "$0")/.."
VERSION="${1:-$(git describe --tags --always --dirty)}"
OUT=dist
rm -rf "$OUT"; mkdir -p "$OUT"

LDFLAGS="-s -w -X main.version=${VERSION}"

build() {
  local goos="$1" goarch="$2" ext="${3:-}"
  local name="cfgtool_${VERSION}_${goos}_${goarch}${ext}"
  echo "сборка $name"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "$OUT/$name" ./cmd/cfgtool
}

build linux   amd64
build windows amd64 .exe

# состав зависимостей, вшитых в бинарник
go version -m "$OUT/cfgtool_${VERSION}_linux_amd64" > "$OUT/SBOM-go-modules.txt"

( cd "$OUT" && sha256sum cfgtool_* > SHA256SUMS.txt )
echo
cat "$OUT/SHA256SUMS.txt"
echo
echo "Выпуск ${VERSION} собран в $OUT/"
