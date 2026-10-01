#!/usr/bin/env bash
# Полный набор проверок продукта.
# Использование: scripts/check.sh [каталог-отчётов]
# По умолчанию — evidence/<текущий-тег-или-коммит>.
# Скрипт не прерывается на находках: цель — собрать полную картину.
set -u
cd "$(dirname "$0")/.."
export PATH="$PATH:$(cygpath -u "$(go env GOPATH)" 2>/dev/null || go env GOPATH)/bin"

LABEL="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo local)}"
DIR="evidence/$LABEL"
mkdir -p "$DIR"
echo "Отчёты: $DIR"
echo

run() {
  local name="$1"; shift
  printf '== %-14s ' "$name"
  "$@" >"$DIR/$name.txt" 2>&1
  echo "код возврата: $?"
}

printf '== %-14s ' gofmt
gofmt -l . > "$DIR/gofmt.txt"
echo "файлов без форматирования: $(wc -l < "$DIR/gofmt.txt")"

run vet          go vet ./...
run test         go test ./... -count=1 -v
run cover        go test ./... -count=1 -coverprofile="$DIR/coverage.out"
run staticcheck  staticcheck ./...
run gosec        gosec -fmt=text -color=false ./...
run govulncheck  govulncheck ./...

if [ -s "$DIR/coverage.out" ]; then
  go tool cover -func="$DIR/coverage.out" | tail -1 | tee "$DIR/coverage-total.txt"
fi
echo
echo "Готово."
