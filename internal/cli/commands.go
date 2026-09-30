package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/Homeveloper/ProjectCreateForSystemSoftware/internal/config"
)

// output описывает общие флаги записи результата.
type output struct {
	name   string
	outDir string
}

func (o *output) bind(fs *flag.FlagSet) {
	fs.StringVar(&o.name, "o", "", "имя файла результата (по умолчанию — стандартный вывод)")
	fs.StringVar(&o.outDir, "out-dir", ".", "каталог, в который записывается результат")
}

// emit записывает результат в файл или в стандартный вывод.
func (o *output) emit(cfg config.Config, fallback config.Format) error {
	format := fallback
	if o.name != "" {
		detected, err := config.DetectFormat(o.name)
		if err != nil {
			return err
		}
		format = detected
	}
	data, err := config.Encode(cfg, format)
	if err != nil {
		return err
	}
	if o.name == "" {
		_, err = os.Stdout.Write(data)
		return err
	}
	path, err := config.Save(o.outDir, o.name, data)
	if err != nil {
		return err
	}
	fmt.Printf("записано: %s\n", path)
	return nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

// parseArgs разбирает флаги, которые могут стоять как до, так и после
// позиционных аргументов: пакет flag из стандартной библиотеки
// самостоятельно прекращает разбор на первом позиционном аргументе.
func parseArgs(fs *flag.FlagSet, args []string) ([]string, error) {
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

func cmdValidate(args []string) error {
	fs := newFlagSet("validate")
	schemaPath := fs.String("schema", "", "путь к файлу схемы в формате JSON")
	verbose := fs.Bool("verbose", false, "выводить подробности разбора")
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return errors.New("использование: cfgtool validate <файл> [--schema <схема.json>]")
	}
	path := positional[0]
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	if *verbose {
		fmt.Fprintf(os.Stderr, "разобран %s: %#v\n", path, map[string]interface{}(cfg))
	}
	if *schemaPath == "" {
		fmt.Printf("%s: синтаксис корректен, схема не задана\n", path)
		return nil
	}
	schema, err := config.LoadSchema(*schemaPath)
	if err != nil {
		return err
	}
	problems := schema.Validate(cfg)
	if len(problems) > 0 {
		for _, problem := range problems {
			fmt.Fprintln(os.Stderr, "  - "+problem)
		}
		return fmt.Errorf("%s: нарушений схемы — %d", path, len(problems))
	}
	fmt.Printf("%s: соответствует схеме %s\n", path, *schemaPath)
	return nil
}

func cmdConvert(args []string) error {
	fs := newFlagSet("convert")
	var out output
	out.bind(fs)
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return errors.New("использование: cfgtool convert <файл> [-o <имя>] [--out-dir <каталог>]")
	}
	path := positional[0]
	sourceFormat, err := config.DetectFormat(path)
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	target := config.FormatJSON
	if sourceFormat == config.FormatJSON {
		target = config.FormatYAML
	}
	return out.emit(cfg, target)
}

func cmdMerge(args []string) error {
	fs := newFlagSet("merge")
	var out output
	out.bind(fs)
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) < 2 {
		return errors.New("использование: cfgtool merge <файл> <файл> [...] [-o <имя>]")
	}
	merged := config.Config{}
	for _, path := range positional {
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}
		merged = config.Merge(merged, cfg)
	}
	baseFormat, err := config.DetectFormat(positional[0])
	if err != nil {
		return err
	}
	return out.emit(merged, baseFormat)
}

func cmdRedact(args []string) error {
	fs := newFlagSet("redact")
	var out output
	out.bind(fs)
	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return errors.New("использование: cfgtool redact <файл> [-o <имя>]")
	}
	path := positional[0]
	sourceFormat, err := config.DetectFormat(path)
	if err != nil {
		return err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	return out.emit(config.Redact(cfg), sourceFormat)
}
