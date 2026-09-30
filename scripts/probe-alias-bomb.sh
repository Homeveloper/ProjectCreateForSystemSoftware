#!/usr/bin/env bash
# Демонстрация усиления при разборе YAML с псевдонимами (alias/anchor).
#
# Файл вида
#   l0: &l0 ["x", ... 9 элементов ...]
#   l1: &l1 [*l0, ... 9 ссылок ...]
#   ...
# при разборе разворачивается в 9^(N+1) значений, тогда как сам файл
# растёт линейно — примерно на 46 байт на уровень.
#
# Скрипт намеренно ограничен малыми уровнями: цель — измерить
# коэффициент усиления, а не исчерпать память машины.
#   - уровни 2..6 (максимум 9^7 = 4 782 969 значений);
#   - на каждый запуск лимит времени;
#   - цикл прекращается, если результат превысил порог размера.
#
# Использование: scripts/probe-alias-bomb.sh [каталог-отчётов]
set -u
cd "$(dirname "$0")/.."

LABEL="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo local)}"
OUT_DIR="reports/$LABEL"
REPORT="$OUT_DIR/dos-alias-bomb.txt"
MAX_LEVEL=6
TIME_LIMIT=30
SIZE_GUARD=$((64 * 1024 * 1024))

mkdir -p "$OUT_DIR"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

go build -o "$WORK/cfgtool" ./cmd/cfgtool || exit 1
export GOMEMLIMIT=1GiB

make_bomb() {
  local levels="$1" out="$2"
  {
    printf 'l0: &l0 ["x","x","x","x","x","x","x","x","x"]\n'
    local i j prev refs
    for ((i = 1; i <= levels; i++)); do
      prev=$((i - 1))
      refs=""
      for ((j = 1; j <= 9; j++)); do refs="$refs*l$prev,"; done
      printf 'l%d: &l%d [%s]\n' "$i" "$i" "${refs%,}"
    done
    printf 'root: *l%d\n' "$levels"
  } > "$out"
}

{
  echo "Демонстрация усиления при разборе YAML с псевдонимами"
  echo "Состояние продукта: $(git describe --tags --always --dirty 2>/dev/null || echo local)"
  echo "Ограничения прогона: уровни 2..$MAX_LEVEL, лимит ${TIME_LIMIT}с на запуск,"
  echo "порог прекращения цикла — $SIZE_GUARD байт результата."
  echo
  printf '%-7s %-10s %-12s %-14s %-10s %s\n' \
    "уровни" "вход, Б" "значений" "результат, Б" "время, мс" "итог"
  printf '%s\n' "-------------------------------------------------------------------------------"
} > "$REPORT"

rejected=0
accepted=0

for ((level = 2; level <= MAX_LEVEL; level++)); do
  bomb="$WORK/bomb$level.yaml"
  make_bomb "$level" "$bomb"
  in_size=$(wc -c < "$bomb")
  values=$((9 ** (level + 1)))

  start=$(date +%s%3N)
  timeout "$TIME_LIMIT" "$WORK/cfgtool" convert "$bomb" \
    > "$WORK/out$level.json" 2> "$WORK/err$level.txt"
  rc=$?
  end=$(date +%s%3N)

  out_size=$(wc -c < "$WORK/out$level.json" 2>/dev/null || echo 0)
  elapsed=$((end - start))

  if [ "$rc" -eq 124 ]; then
    outcome="прервано по лимиту времени"
  elif [ "$rc" -ne 0 ]; then
    rejected=$((rejected + 1))
    outcome="отклонено: $(head -1 "$WORK/err$level.txt" | sed 's/^cfgtool: //')"
  else
    accepted=$((accepted + 1))
    outcome="принято, усиление x$((out_size / in_size))"
  fi

  printf '%-7s %-10s %-12s %-14s %-10s %s\n' \
    "$level" "$in_size" "$values" "$out_size" "$elapsed" "$outcome" >> "$REPORT"
  echo "уровень $level: $outcome"

  [ "$rc" -eq 124 ] && break
  if [ "$out_size" -gt "$SIZE_GUARD" ]; then
    echo "порог размера превышен, дальнейшие уровни не запускаются" >> "$REPORT"
    echo "порог размера превышен, останов" >&2
    break
  fi
done

{
  echo
  echo "Принято уровней: $accepted, отклонено: $rejected."
  echo
  echo "Как читать. Объём разбираемых данных растёт примерно в 9 раз на каждую"
  echo "добавленную строку, а сам файл — примерно на 46 байт. Если продукт"
  echo "принимает такой файл, объём работы ограничен только памятью машины:"
  echo "уровень 9 соответствует примерно 3.9e9 значений, уровень 10 — 3.5e10."
  echo "Если продукт отклоняет файл, в колонке итога указана причина отказа."
} >> "$REPORT"

echo
cat "$REPORT"
