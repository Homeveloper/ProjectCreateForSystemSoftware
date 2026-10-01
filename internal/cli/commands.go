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
	force  bool
}

func (o *output) bind(fs *flag.FlagSet) {
	fs.StringVar(&o.name, "o", "", "имя файла результата (по умолчанию — стандартный вывод)")
	fs.StringVar(&o.outDir, "out-dir", ".", "каталог, в который записывается результат")
	fs.BoolVar(&o.force, "force", false, "разрешить замену существующего файла результата")
}

// emit записывает результат в файл или в стандартный вывод.
func (o *output) emit(cfg config.Config, fallback config.Format) error {
	format := fallback
	if o.name != "" {
		detected, err := config.DetectFormat(o.name)
		if err != nil {
			return withCode(ExitUsage, err)
		}
		format = detected
	}
	data, err := config.Encode(cfg, format)
	if err != nil {
		return withCode(ExitOutput, err)
	}
	if o.name == "" {
		if _, err := os.Stdout.Write(data); err != nil {
			return withCode(ExitOutput, err)
		}
		return nil
	}
	path, err := config.SaveWithOptions(o.outDir, o.name, data,
		config.SaveOptions{Overwrite: o.force})
	if err != nil {
		return withCode(ExitOutput, err)
	}
	fmt.Printf("записано: %s\n", path)
	return nil
}

// limitFlags задают границы объёма и сложности входных данных.
type limitFlags struct {
	maxBytes int64
	maxDepth int
}

func (l *limitFlags) bind(fs *flag.FlagSet) {
	fs.Int64Var(&l.maxBytes, "max-bytes", config.DefaultLimits.MaxBytes,
		"предельный размер входного файла в байтах")
	fs.IntVar(&l.maxDepth, "max-depth", config.DefaultLimits.MaxDepth,
		"предельная глубина вложенности структур")
}

func (l limitFlags) limits() config.Limits {
	return config.Limits{MaxBytes: l.maxBytes, MaxDepth: l.maxDepth}
}

// load читает конфигурацию, помечая неудачу кодом входной ошибки.
func (l limitFlags) load(path string) (config.Config, error) {
	cfg, err := config.LoadWithLimits(path, l.limits())
	if err != nil {
		return nil, withCode(ExitInput, err)
	}
	return cfg, nil
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
			return nil, withCode(ExitUsage, err)
		}
		rest := fs.Args()
		if len(rest) == 0 {
			return positional, nil
		}
		positional = append(positional, rest[0])
		args = rest[1:]
	}
}

func usageError(text string) error {
	return withCode(ExitUsage, errors.New(text))
}

func cmdValidate(args []string) error {
	fs := newFlagSet("validate")
	schemaPath := fs.String("schema", "", "путь к файлу схемы в формате JSON")
	verbose := fs.Bool("verbose", false, "выводить подробности разбора")
	var lim limitFlags
	lim.bind(fs)

	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usageError("использование: cfgtool validate <файл> [--schema <схема.json>]")
	}
	path := positional[0]

	cfg, err := lim.load(path)
	if err != nil {
		return err
	}
	if *verbose {
		// Выводится копия со скрытыми секретами: подробный режим
		// обычно попадает в журнал сборки.
		fmt.Fprintf(os.Stderr, "разобран %s: %#v\n", path,
			map[string]interface{}(config.Redact(cfg)))
	}
	if *schemaPath == "" {
		fmt.Printf("%s: синтаксис корректен, схема не задана\n", path)
		return nil
	}

	schema, err := config.LoadSchema(*schemaPath)
	if err != nil {
		return withCode(ExitInput, err)
	}
	problems := schema.Validate(cfg)
	if len(problems) > 0 {
		for _, problem := range problems {
			fmt.Fprintln(os.Stderr, "  - "+problem)
		}
		return withCode(ExitSchema,
			fmt.Errorf("%s: нарушений схемы — %d", path, len(problems)))
	}
	fmt.Printf("%s: соответствует схеме %s\n", path, *schemaPath)
	return nil
}

// cmdSummary — режим краткого просмотра: ничего не записывает
// и не выводит значений полей.
func cmdSummary(args []string) error {
	fs := newFlagSet("summary")
	var lim limitFlags
	lim.bind(fs)

	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usageError("использование: cfgtool summary <файл>")
	}
	path := positional[0]

	format, err := config.DetectFormat(path)
	if err != nil {
		return withCode(ExitUsage, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return withCode(ExitInput, fmt.Errorf("чтение сведений о %s: %w", path, err))
	}
	cfg, err := lim.load(path)
	if err != nil {
		return err
	}

	summary := config.Summarize(cfg)
	fmt.Printf("%s\n", path)
	fmt.Printf("  формат:                 %s\n", format)
	fmt.Printf("  размер:                 %d байт\n", info.Size())
	fmt.Printf("  полей верхнего уровня:  %d\n", summary.TopLevelKeys)
	fmt.Printf("  всего полей:            %d\n", summary.Fields)
	fmt.Printf("  конечных значений:      %d\n", summary.Leaves)
	fmt.Printf("  глубина вложенности:    %d\n", summary.MaxDepth)
	fmt.Printf("  секретных полей:        %d\n", len(summary.SecretFields))
	for _, field := range summary.SecretFields {
		fmt.Printf("    %s\n", field)
	}
	return nil
}

func cmdConvert(args []string) error {
	fs := newFlagSet("convert")
	var out output
	out.bind(fs)
	var lim limitFlags
	lim.bind(fs)

	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usageError("использование: cfgtool convert <файл> [-o <имя>] [--out-dir <каталог>] [--force]")
	}
	path := positional[0]

	sourceFormat, err := config.DetectFormat(path)
	if err != nil {
		return withCode(ExitUsage, err)
	}
	cfg, err := lim.load(path)
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
	var lim limitFlags
	lim.bind(fs)

	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) < 2 {
		return usageError("использование: cfgtool merge <файл> <файл> [...] [-o <имя>] [--force]")
	}

	merged := config.Config{}
	for _, path := range positional {
		cfg, err := lim.load(path)
		if err != nil {
			return err
		}
		merged = config.Merge(merged, cfg)
	}
	baseFormat, err := config.DetectFormat(positional[0])
	if err != nil {
		return withCode(ExitUsage, err)
	}
	return out.emit(merged, baseFormat)
}

func cmdRedact(args []string) error {
	fs := newFlagSet("redact")
	var out output
	out.bind(fs)
	var lim limitFlags
	lim.bind(fs)

	positional, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(positional) != 1 {
		return usageError("использование: cfgtool redact <файл> [-o <имя>] [--force]")
	}
	path := positional[0]

	sourceFormat, err := config.DetectFormat(path)
	if err != nil {
		return withCode(ExitUsage, err)
	}
	cfg, err := lim.load(path)
	if err != nil {
		return err
	}
	return out.emit(config.Redact(cfg), sourceFormat)
}
