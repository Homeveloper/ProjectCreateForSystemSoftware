# cfgtool

Консольная утилита для проверки и преобразования конфигураций в форматах YAML и JSON.

Продукт создан как учебный для курса по безопасной разработке (вариант 8 —
«CLI для обработки конфигураций»).

## Возможности

| Команда | Назначение |
|---|---|
| `validate <файл> [--schema <схема.json>]` | проверка синтаксиса и соответствия схеме |
| `convert <файл> [-o <имя>]` | преобразование YAML в JSON и обратно |
| `merge <файл> <файл> [...] [-o <имя>]` | объединение конфигураций по порядку |
| `redact <файл> [-o <имя>]` | скрытие значений секретных полей |
| `version` | версия сборки |
| `help` | справка |

Если `-o` не указан, результат выводится в стандартный поток вывода.
Каталог для записи задаётся флагом `--out-dir` (по умолчанию — текущий).

## Сборка и запуск

```
go build -o bin/cfgtool ./cmd/cfgtool
./bin/cfgtool validate testdata/app.yaml --schema testdata/schema.json
```

## Примеры

```
# проверить конфигурацию по схеме
cfgtool validate testdata/app.yaml --schema testdata/schema.json

# преобразовать YAML в JSON в каталог build
cfgtool convert testdata/app.yaml -o app.json --out-dir build

# наложить окружение на базовую конфигурацию
cfgtool merge testdata/app.yaml testdata/override.yaml -o effective.yaml --out-dir build

# получить копию без секретов, пригодную для передачи
cfgtool redact testdata/app.yaml -o app.public.yaml --out-dir build
```

## Поддерживаемое подмножество схемы

Схема задаётся файлом JSON и поддерживает ключевые слова
`type`, `required`, `properties`, `items`, `enum`, `pattern`,
`minimum`, `maximum`. Пример — [testdata/schema.json](testdata/schema.json).

## Разработка

```
go test ./...      # тесты
go vet ./...       # статические проверки компилятора
gofmt -l .         # форматирование
```

Проверки безопасности, их результаты и решение о выпуске описаны
в [project.md](project.md).
