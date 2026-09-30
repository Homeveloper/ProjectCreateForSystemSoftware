#!/usr/bin/env bash
# Полный набор проверок продукта. Отчёты складываются в reports/.
# Скрипт не прерывается на находках: цель — собрать полную картину.
set -u
cd "$(dirname "$0")/.."
export PATH="$PATH:$(cygpath -u "$(go env GOPATH)" 2>/dev/null || go env GOPATH)/bin"
mkdir -p reports

run() {
  local name="$1"; shift
  echo "== $name"
  "$@" >"reports/$name.txt" 2>&1
  echo "   код возврата: $? -> reports/$name.txt"
}

echo "== gofmt"
gofmt -l . | tee reports/gofmt.txt
run vet        go vet ./...
run test       go test ./... -count=1 -v
run cover      go test ./... -count=1 -coverprofile=reports/coverage.out
run staticcheck staticcheck ./...
run gosec      gosec -fmt=text ./...
run govulncheck govulncheck ./...

if [ -f reports/coverage.out ]; then
  go tool cover -func=reports/coverage.out | tail -1 | tee reports/coverage-total.txt
fi
echo
echo "Отчёты готовы: reports/"
