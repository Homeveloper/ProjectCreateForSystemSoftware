package cli

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	yamlFixture   = "../../testdata/app.yaml"
	schemaFixture = "../../testdata/schema.json"
)

func TestRunRequiresCommand(t *testing.T) {
	if err := Run(nil, "test"); err == nil {
		t.Error("ожидалась ошибка при отсутствии команды")
	}
}

func TestRunUnknownCommand(t *testing.T) {
	if err := Run([]string{"frobnicate"}, "test"); err == nil {
		t.Error("ожидалась ошибка для неизвестной команды")
	}
}

func TestRunVersionAndHelp(t *testing.T) {
	if err := Run([]string{"version"}, "test"); err != nil {
		t.Errorf("version: %v", err)
	}
	if err := Run([]string{"help"}, "test"); err != nil {
		t.Errorf("help: %v", err)
	}
}

func TestRunValidateWithSchema(t *testing.T) {
	if err := Run([]string{"validate", yamlFixture, "--schema", schemaFixture}, "test"); err != nil {
		t.Errorf("validate: %v", err)
	}
}

func TestRunValidateReportsSchemaProblems(t *testing.T) {
	err := Run([]string{"validate", "../../testdata/invalid.yaml", "--schema", schemaFixture}, "test")
	if err == nil {
		t.Error("ожидалась ошибка для конфигурации, нарушающей схему")
	}
}

func TestRunConvertWritesFile(t *testing.T) {
	dir := t.TempDir()
	if err := Run([]string{"convert", yamlFixture, "-o", "app.json", "--out-dir", dir}, "test"); err != nil {
		t.Fatalf("convert: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "app.json")); err != nil {
		t.Fatalf("результат не создан: %v", err)
	}
}

func TestRunMergeRequiresTwoFiles(t *testing.T) {
	if err := Run([]string{"merge", yamlFixture}, "test"); err == nil {
		t.Error("ожидалась ошибка: merge требует минимум два файла")
	}
}

func TestRunRedactWritesFile(t *testing.T) {
	dir := t.TempDir()
	if err := Run([]string{"redact", yamlFixture, "-o", "safe.yaml", "--out-dir", dir}, "test"); err != nil {
		t.Fatalf("redact: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "safe.yaml")); err != nil {
		t.Fatalf("результат не создан: %v", err)
	}
}
