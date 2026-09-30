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
  convert  <файл> [-o <имя>] [--out-dir <кат>] преобразовать между YAML и JSON
  merge    <файл> <файл> [...] [-o <имя>]      объединить конфигурации по порядку
  redact   <файл> [-o <имя>]                   скрыть значения секретных полей
  version                                      вывести версию
  help                                         вывести эту справку

Если -o не задан, результат выводится в стандартный поток вывода.
`

// Run выполняет команду, переданную в аргументах.
func Run(args []string, version string) error {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		return errors.New("команда не указана")
	}
	switch args[0] {
	case "validate":
		return cmdValidate(args[1:])
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
		return fmt.Errorf("неизвестная команда %q", args[0])
	}
}
