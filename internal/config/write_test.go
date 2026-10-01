package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveRefusesExistingFile(t *testing.T) {
	dir := t.TempDir()
	if _, err := Save(dir, "out.json", []byte("первое\n")); err != nil {
		t.Fatalf("первая запись: %v", err)
	}

	_, err := Save(dir, "out.json", []byte("второе\n"))
	if err == nil {
		t.Fatal("повторная запись выполнена; ожидался отказ без --force")
	}
	var existsErr *ErrOutputExists
	if !errors.As(err, &existsErr) {
		t.Errorf("ожидалась ошибка ErrOutputExists, получено: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "out.json"))
	if err != nil {
		t.Fatalf("чтение: %v", err)
	}
	if string(data) != "первое\n" {
		t.Errorf("содержимое изменено: %q", data)
	}
}

func TestSaveReplacesFileWithOverwrite(t *testing.T) {
	dir := t.TempDir()
	if _, err := Save(dir, "out.json", []byte("первое\n")); err != nil {
		t.Fatalf("первая запись: %v", err)
	}
	if _, err := SaveWithOptions(dir, "out.json", []byte("второе\n"),
		SaveOptions{Overwrite: true}); err != nil {
		t.Fatalf("замена: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "out.json"))
	if err != nil {
		t.Fatalf("чтение: %v", err)
	}
	if string(data) != "второе\n" {
		t.Errorf("содержимое = %q, ожидалось второе", data)
	}
}

// Запись выполняется через временный файл; после успешного завершения
// в каталоге не должно оставаться ничего лишнего.
func TestSaveLeavesNoTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	if _, err := Save(dir, "out.json", []byte("{}\n")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "out.json" {
		var names []string
		for _, entry := range entries {
			names = append(names, entry.Name())
		}
		t.Errorf("в каталоге остались лишние файлы: %v", names)
	}
}

func TestSaveRefusesDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "busy"), 0o700); err != nil {
		t.Fatalf("подготовка: %v", err)
	}
	if _, err := SaveWithOptions(dir, "busy", []byte("{}\n"),
		SaveOptions{Overwrite: true}); err == nil {
		t.Error("запись поверх каталога выполнена; ожидался отказ")
	}
}

func TestSummarizeCountsWithoutRevealingValues(t *testing.T) {
	cfg, err := Load(yamlFixture)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	summary := Summarize(cfg)

	if summary.TopLevelKeys != 4 {
		t.Errorf("полей верхнего уровня = %d, ожидалось 4", summary.TopLevelKeys)
	}
	if summary.MaxDepth < 1 {
		t.Errorf("глубина = %d, ожидалось не меньше 1", summary.MaxDepth)
	}
	if len(summary.SecretFields) != 2 {
		t.Errorf("секретных полей = %d, ожидалось 2: %v",
			len(summary.SecretFields), summary.SecretFields)
	}
	joined := strings.Join(summary.SecretFields, "\n")
	if strings.Contains(joined, secretValue) {
		t.Errorf("описание содержит значение секретного поля:\n%s", joined)
	}
	for _, want := range []string{"password", "api_token"} {
		if !strings.Contains(joined, want) {
			t.Errorf("в описании нет пути поля %q:\n%s", want, joined)
		}
	}
}
