// Package cli разбирает аргументы командной строки и выполняет команды cfgtool.
package cli

import (
	"errors"
	"fmt"
	"os"
)

const usage = `cfgtool — проверка и преобразование конфигураций YAML и JSON.

Использование:
  cfgtool <команда> [флаги] [аргументы]

Команды:
  validate <файл> [--schema <схема.json>]      проверить синтаксис и соответствие схеме
  summary  <файл>                              краткое описание без записи результата
  convert  <файл> [-o <имя>] [--out-dir <кат>] преобразовать между YAML и JSON
  merge    <файл> <файл> [...] [-o <имя>]      объединить конфигурации по порядку
  redact   <файл> [-o <имя>]                   скрыть значения секретных полей
  version                                      вывести версию
  help                                         вывести эту справку

Общие флаги:
  -o <имя>            файл результата; без него результат идёт в стандартный вывод
  --out-dir <каталог> каталог результата, выход за его пределы отклоняется
  --force             разрешить замену существующего файла результата
  --max-bytes <N>     предельный размер входного файла, по умолчанию 10485760
  --max-depth <N>     предельная глубина вложенности, по умолчанию 64

Коды завершения:
  0  успешно
  1  ошибка в аргументах командной строки
  2  входной файл не прочитан или не разобран
  3  конфигурация нарушает схему
  4  результат не записан
`

// Run выполняет команду, переданную в аргументах.
func Run(args []string, version string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return withCode(ExitUsage, errors.New("команда не указана"))
	}
	switch args[0] {
	case "validate":
		return cmdValidate(args[1:])
	case "summary":
		return cmdSummary(args[1:])
	case "convert":
		return cmdConvert(args[1:])
	case "merge":
		return cmdMerge(args[1:])
	case "redact":
		return cmdRedact(args[1:])
	case "version":
		fmt.Println("cfgtool " + version)
		return nil
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		fmt.Fprint(os.Stderr, usage)
		return withCode(ExitUsage, fmt.Errorf("неизвестная команда %q", args[0]))
	}
}
