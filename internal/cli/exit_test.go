package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// captureStdout перехватывает стандартный поток вывода на время вызова fn.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = original }()

	collected := make(chan string, 1)
	go func() {
		var buffer bytes.Buffer
		_, _ = io.Copy(&buffer, reader)
		collected <- buffer.String()
	}()

	fn()

	if err := writer.Close(); err != nil {
		t.Fatalf("закрытие потока: %v", err)
	}
	return <-collected
}

// Коды завершения различают причины отказа: вызывающий сценарий
// должен реагировать на них, не разбирая текст сообщения.
func TestExitCodesDistinguishFailures(t *testing.T) {
	dir := t.TempDir()

	cases := []struct {
		name string
		args []string
		want int
	}{
		{
			name: "успех",
			args: []string{"validate", yamlFixture, "--schema", schemaFixture},
			want: ExitOK,
		},
		{
			name: "неизвестная команда",
			args: []string{"frobnicate"},
			want: ExitUsage,
		},
		{
			name: "нет обязательного аргумента",
			args: []string{"validate"},
			want: ExitUsage,
		},
		{
			name: "входной файл отсутствует",
			args: []string{"validate", filepath.Join(dir, "нет-такого.yaml")},
			want: ExitInput,
		},
		{
			name: "нарушение схемы",
			args: []string{"validate", "../../testdata/invalid.yaml", "--schema", schemaFixture},
			want: ExitSchema,
		},
		{
			name: "результат за пределами каталога",
			args: []string{"convert", yamlFixture, "-o", "../escaped.json", "--out-dir", dir},
			want: ExitOutput,
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var err error
			_ = captureStdout(t, func() { err = Run(test.args, "test") })
			if got := ExitCodeFor(err); got != test.want {
				t.Errorf("код завершения = %d, ожидался %d (ошибка: %v)", got, test.want, err)
			}
		})
	}
}

// Повторная запись в существующий файл отклоняется, пока не указан --force.
func TestConvertRefusesToOverwriteWithoutForce(t *testing.T) {
	dir := t.TempDir()

	if err := Run([]string{"convert", yamlFixture, "-o", "app.json", "--out-dir", dir}, "test"); err != nil {
		t.Fatalf("первая запись: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(dir, "app.json"))
	if err != nil {
		t.Fatalf("чтение: %v", err)
	}

	err = Run([]string{"convert", yamlFixture, "-o", "app.json", "--out-dir", dir}, "test")
	if err == nil {
		t.Fatal("повторная запись выполнена; ожидался отказ без --force")
	}
	if code := ExitCodeFor(err); code != ExitOutput {
		t.Errorf("код завершения = %d, ожидался %d", code, ExitOutput)
	}

	if err := Run([]string{"convert", yamlFixture, "-o", "app.json", "--out-dir", dir, "--force"}, "test"); err != nil {
		t.Errorf("запись с --force: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(dir, "app.json"))
	if err != nil {
		t.Fatalf("чтение: %v", err)
	}
	if string(before) != string(after) {
		t.Error("содержимое после замены отличается от исходного преобразования")
	}
}

// Режим краткого просмотра ничего не записывает и не раскрывает значений.
func TestSummaryWritesNothingAndHidesValues(t *testing.T) {
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(cwd) }()

	source := filepath.Join(cwd, yamlFixture)
	output := captureStdout(t, func() {
		if err := Run([]string{"summary", source}, "test"); err != nil {
			t.Errorf("summary: %v", err)
		}
	})

	if strings.Contains(output, secretValue) {
		t.Errorf("краткое описание содержит значение секретного поля:\n%s", output)
	}
	if !strings.Contains(output, "секретных полей") {
		t.Errorf("в описании нет сведений о секретных полях:\n%s", output)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("режим просмотра создал файлы: %d", len(entries))
	}
}
