#!/usr/bin/env bash
# Воспроизведение проверок исходного состояния продукта.
#
# Код продукта берётся из тега v0.1.0, проверочная оснастка — текущая.
# Так и были получены отчёты в evidence/baseline-v0.1.0/: негативные тесты
# и скрипты появились уже после фиксации исходного состояния, но выполнялись
# против того же кода продукта. Оснастка меняться может, продукт — нет:
# иначе сравнение "до и после" теряет смысл.
#
# Результат складывается в evidence/baseline-v0.1.0-reproduced/,
# чтобы его можно было сравнить с сохранёнными отчётами.
set -u

cd "$(dirname "$0")/.."
ROOT="$(pwd)"
BASELINE_TAG="v0.1.0"
LABEL="baseline-v0.1.0"
TARGET="$ROOT/evidence/${LABEL}-reproduced"

if ! git rev-parse --verify --quiet "$BASELINE_TAG" >/dev/null; then
  echo "тег $BASELINE_TAG не найден" >&2
  exit 1
fi

WORKTREE="$(mktemp -d)/baseline"
cleanup() {
  git worktree remove --force "$WORKTREE" >/dev/null 2>&1 || true
}
trap cleanup EXIT

echo "== разворачиваю продукт из тега $BASELINE_TAG"
git worktree add --detach "$WORKTREE" "$BASELINE_TAG" >/dev/null || exit 1

echo "== переношу текущую проверочную оснастку"
cp "$ROOT/internal/config/security_test.go" "$WORKTREE/internal/config/"
cp "$ROOT/internal/cli/security_test.go" "$WORKTREE/internal/cli/"
cp "$ROOT/internal/cli/cli_test.go" "$WORKTREE/internal/cli/"
mkdir -p "$WORKTREE/scripts"
cp "$ROOT/scripts/check.sh" "$WORKTREE/scripts/"
cp "$ROOT/scripts/probe-alias-bomb.sh" "$WORKTREE/scripts/"

echo "== прогоняю проверки"
cd "$WORKTREE"
bash scripts/check.sh "$LABEL"
bash scripts/probe-alias-bomb.sh "$LABEL" >/dev/null

mkdir -p "$TARGET"
cp -r "$WORKTREE/evidence/$LABEL/." "$TARGET/"

echo
echo "Отчёты: evidence/${LABEL}-reproduced/"
echo
echo "Ожидаемый результат на исходном состоянии:"
echo "  - govulncheck: три уязвимости в gopkg.in/yaml.v2 v2.2.2"
echo "  - gosec: четыре находки"
echo "  - негативные тесты безопасности: восемь падений"
echo "  - демонстрация усиления: файл в 332 байта раскрывается в 251 МБ"
