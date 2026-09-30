package cli

// Тесты в этом файле проверяют свойства безопасности на уровне команд.
// На исходном состоянии продукта (v0.1.0) они не проходят.

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// secretValue — значение из testdata/app.yaml, которое не должно
// появляться в выводе команд.
const secretValue = "s3cr3t-db-password"

// captureStderr перехватывает стандартный поток ошибок на время вызова fn.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()

	original := os.Stderr
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stderr = writer
	defer func() { os.Stderr = original }()

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

// Разрыв 9: подробный режим печатает разобранную конфигурацию целиком,
// включая пароли и маркеры доступа.
func TestVerboseOutputDoesNotLeakSecrets(t *testing.T) {
	output := captureStderr(t, func() {
		if err := Run([]string{"validate", yamlFixture, "--verbose"}, "test"); err != nil {
			t.Errorf("validate: %v", err)
		}
	})

	if strings.Contains(output, secretValue) {
		t.Errorf("подробный вывод содержит значение секретного поля:\n%s", output)
	}
}

// Разрыв 10: флаг -o позволяет записать результат за пределы --out-dir.
func TestConvertRefusesOutputOutsideOutDir(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("подготовка каталога: %v", err)
	}
	escaped := filepath.Join(base, "escaped.json")

	err := Run([]string{"convert", yamlFixture, "-o", "../escaped.json", "--out-dir", root}, "test")
	if err == nil {
		t.Error("результат записан за пределы каталога --out-dir, ожидался отказ")
	}
	if _, statErr := os.Stat(escaped); statErr == nil {
		t.Errorf("файл создан за пределами каталога --out-dir: %s", escaped)
	}
}
