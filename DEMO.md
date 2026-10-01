# Как пощупать продукт самостоятельно

Пошаговая проверка `cfgtool` руками. Выполняется за 15–20 минут,
закрывает всё, что карточка проекта требует показать на защите.

Команды даны для **PowerShell** (то, что открывается в Windows по умолчанию).
Если предпочитаете **Git Bash** — команды те же, но вместо `.\bin\cfgtool.exe`
пишите `./bin/cfgtool.exe`, а вместо `$LASTEXITCODE` — `$?`.

> PowerShell 5.1 не понимает `&&`. Каждую команду запускайте отдельной
> строкой, не склеивайте их.

## Подготовка

```powershell
cd "D:\Учёба\ProjectCreateForSystemSoftware"
go build -o bin\cfgtool.exe .\cmd\cfgtool
.\bin\cfgtool.exe version
```

Ожидается: `cfgtool dev` (или номер версии, если собирали через `scripts/release.sh`).

Создайте временный каталог для результатов, чтобы не мусорить в репозитории:

```powershell
New-Item -ItemType Directory -Force demo-out
```

---

## 1. Корректная конфигурация

Посмотрите, с чем работаем:

```powershell
Get-Content testdata\app.yaml
```

Обратите внимание на два поля: `database.password` и `auth.api_token`.
Дальше мы будем следить, не вылезут ли они куда не надо.

### Проверка по схеме

```powershell
.\bin\cfgtool.exe validate testdata\app.yaml --schema testdata\schema.json
$LASTEXITCODE
```

Ожидается: `соответствует схеме`, код **0**.

### Краткий просмотр

```powershell
.\bin\cfgtool.exe summary testdata\app.yaml
```

Ожидается таблица с форматом, размером, числом полей и глубиной.
**Главное:** в строке «секретных полей» показаны только пути
(`$.auth.api_token`, `$.database.password`), но не значения.
Команда ничего не записывает — убедитесь:

```powershell
Get-ChildItem demo-out
```

Каталог пуст.

---

## 2. Преобразование, слияние, маскирование

```powershell
.\bin\cfgtool.exe convert testdata\app.yaml -o app.json --out-dir demo-out
Get-Content demo-out\app.json
```

```powershell
.\bin\cfgtool.exe merge testdata\app.yaml testdata\override.yaml
```

Сравните с `testdata\override.yaml`: должны замениться `service.port` на 9090,
`service.mode` на `staging`, `database.host` на `db.staging.internal`.
Остальное сохраняется из базового файла.

```powershell
.\bin\cfgtool.exe redact testdata\app.yaml
```

Ожидается: вместо пароля и маркера доступа — `***REDACTED***`,
остальные поля не тронуты.

---

## 3. Реакция на плохие данные

### Повреждённый файл

```powershell
Set-Content -Encoding utf8 demo-out\broken.yaml "database:`n  password: s3cr3t-db-password`n  broken: [unclosed"
.\bin\cfgtool.exe validate demo-out\broken.yaml
$LASTEXITCODE
```

Ожидается код **2**. Сообщение называет номер строки и объём файла.

**Теперь главное — проверьте, не утёк ли пароль:**

```powershell
.\bin\cfgtool.exe validate demo-out\broken.yaml 2>&1 | Select-String "s3cr3t"
```

Ожидается: **пусто**. Содержимое файла в сообщение об ошибке не попадает.

### Нарушение схемы

```powershell
Get-Content testdata\invalid.yaml
.\bin\cfgtool.exe validate testdata\invalid.yaml --schema testdata\schema.json
$LASTEXITCODE
```

Ожидается код **3** и четыре замечания с путями: шаблон имени, предел порта,
недопустимое значение `mode`, отсутствие обязательного поля `database`.

### Нарушение на секретном поле

```powershell
Set-Content -Encoding utf8 demo-out\secret.yaml "password: s3cr3t-db-password"
Set-Content -Encoding utf8 demo-out\secret-schema.json '{"type":"object","properties":{"password":{"type":"string","pattern":"^[A-Z]+$"}}}'
.\bin\cfgtool.exe validate demo-out\secret.yaml --schema demo-out\secret-schema.json
```

Ожидается: `$.password: значение ***REDACTED*** не соответствует шаблону "^[A-Z]+$"`.

Это ключевой момент: путь поля и нарушенное правило видны — диагностика
осталась полезной, — а значение заменено маской.

---

## 4. Граничные условия

### Существующий файл результата

`demo-out\app.json` мы создали на шаге 2. Попробуйте записать поверх:

```powershell
.\bin\cfgtool.exe convert testdata\app.yaml -o app.json --out-dir demo-out
$LASTEXITCODE
```

Ожидается код **4** и отказ с подсказкой про `--force`. Старый файл цел.

```powershell
.\bin\cfgtool.exe convert testdata\app.yaml -o app.json --out-dir demo-out --force
$LASTEXITCODE
```

Теперь код **0**, файл заменён.

### Выход за пределы каталога

```powershell
.\bin\cfgtool.exe convert testdata\app.yaml -o "..\escaped.json" --out-dir demo-out
.\bin\cfgtool.exe convert testdata\app.yaml -o "sub\..\..\escaped.json" --out-dir demo-out
.\bin\cfgtool.exe convert testdata\app.yaml -o "C:\escaped.json" --out-dir demo-out
```

Все три отклоняются. Убедитесь, что ничего не создалось:

```powershell
Test-Path ..\escaped.json
Test-Path C:\escaped.json
```

Оба ответа — `False`.

### Пределы размера и глубины

Слишком большой файл:

```powershell
$big = "key: """ + ("x" * 20000000) + """"
Set-Content -Encoding ascii demo-out\huge.yaml $big
.\bin\cfgtool.exe validate demo-out\huge.yaml
$LASTEXITCODE
```

Ожидается код **2** и сообщение про предел 10485760 байт.

Предел настраивается — поднимите его, и файл примется:

```powershell
.\bin\cfgtool.exe summary demo-out\huge.yaml --max-bytes 25000000
$LASTEXITCODE
```

Слишком глубокая вложенность:

```powershell
$deep = ("{a: " * 5000) + "1" + ("}" * 5000)
Set-Content -Encoding ascii demo-out\deep.yaml $deep
.\bin\cfgtool.exe validate demo-out\deep.yaml
$LASTEXITCODE
```

Ожидается код **2** и сообщение про предел глубины 64.

---

## 5. Главное: сравнение с исходным состоянием

Это центральный номер защиты. Собираем бинарник из тега `v0.1.0`
и натравливаем обе версии на один и тот же файл.

```powershell
git worktree add --detach ..\cfgtool-baseline v0.1.0
cd ..\cfgtool-baseline
go build -o ..\ProjectCreateForSystemSoftware\bin\cfgtool-old.exe .\cmd\cfgtool
cd ..\ProjectCreateForSystemSoftware
```

Посмотрите на входной файл — 332 байта:

```powershell
Get-Content testdata\alias-bomb.yaml
(Get-Item testdata\alias-bomb.yaml).Length
```

### Исходное состояние

```powershell
Measure-Command { .\bin\cfgtool-old.exe convert testdata\alias-bomb.yaml > demo-out\bomb-old.json }
$LASTEXITCODE
(Get-Item demo-out\bomb-old.json).Length
```

Ожидается: несколько секунд, код **0** (!), результат около **251 МБ**.
Старая версия считает это нормальной работой.

### Выпуск

```powershell
Measure-Command { .\bin\cfgtool.exe convert testdata\alias-bomb.yaml > demo-out\bomb-new.json }
$LASTEXITCODE
.\bin\cfgtool.exe convert testdata\alias-bomb.yaml
```

Ожидается: код **2**, пустой результат, сообщение
`yaml: document contains excessive aliasing`.

> `Measure-Command` включает в замер запуск процесса, поэтому абсолютные
> числа будут заметно больше чистого времени работы. Сравнивайте две
> величины между собой, а не с секундомером: у исходного состояния время
> растёт с объёмом разворачиваемых данных, у выпуска — нет.
> Главные признаки всё равно другие: **код завершения** и **размер
> результата**.

Уберите за собой:

```powershell
git worktree remove --force ..\cfgtool-baseline
Remove-Item bin\cfgtool-old.exe
Remove-Item demo-out\bomb-old.json
```

---

## 6. Целостность артефакта выпуска

Сборка выпуска выполняется скриптом и требует Git Bash:

```
bash scripts/release.sh v1.1.0
```

Затем в PowerShell:

```powershell
Get-ChildItem dist
Get-Content dist\SHA256SUMS.txt
```

Проверьте, что суммы сходятся:

```powershell
Get-FileHash dist\cfgtool_v1.1.0_windows_amd64.exe -Algorithm SHA256
```

Сравните значение `Hash` со строкой из `SHA256SUMS.txt` (регистр не совпадает —
PowerShell печатает заглавными, это нормально).

Посмотрите, с каким состоянием кода связан бинарник:

```powershell
go version -m dist\cfgtool_v1.1.0_linux_amd64 | Select-String "mod|dep|vcs"
```

Ожидается версия модуля `v1.1.0`, единственная зависимость `yaml.v3 v3.0.1`,
номер коммита и `vcs.modified=false`.

Убедитесь, что сборка воспроизводима — соберите второй раз и сравните:

```powershell
Copy-Item dist\SHA256SUMS.txt demo-out\first.txt
bash scripts/release.sh v1.1.0
Compare-Object (Get-Content demo-out\first.txt) (Get-Content dist\SHA256SUMS.txt)
```

Ожидается: **пустой вывод** — значит суммы совпали.

---

## 7. Полный прогон проверок

Требует Git Bash и установленных инструментов:

```
go install golang.org/x/vuln/cmd/govulncheck@v1.8.0
go install honnef.co/go/tools/cmd/staticcheck@v0.8.1
go install github.com/securego/gosec/v2/cmd/gosec@v2.29.0
```

Проверки состояния выпуска:

```
bash scripts/check.sh my-run
```

Отчёты лягут в `evidence/my-run/`. Ожидается: `govulncheck` без находок,
`gosec` две принятые находки G304, `staticcheck` и `go vet` без замечаний,
тесты без падений.

Воспроизведение проверок исходного состояния:

```
bash scripts/reproduce-baseline.sh
```

Ожидается: три уязвимости `govulncheck`, четыре находки `gosec`,
**восемь падений** негативных тестов. Сверьте с сохранёнными отчётами
в `evidence/baseline-v0.1.0/` — должно совпасть всё, кроме времени.

---

## Уборка

```powershell
Remove-Item -Recurse -Force demo-out
```

---

## Чего этой проверкой не увидеть

Три теста на Windows пропускаются и выполняются только в конвейере на Linux:

* два на права доступа POSIX (`0600` на файл, `0700` на каталог) —
  на Windows права POSIX не применяются;
* один на отказ от записи через символьную ссылку — создание ссылки
  на Windows требует прав администратора.

Их результат смотрите во вкладке **Actions** на GitHub, в задании
«Сборка и тесты».

Там же видно, что запуск на коммите `d057a8a` красный — это сделано
намеренно: в том коммите негативные тесты уже добавлены, а улучшение
ещё нет. Следующий запуск зелёный.
