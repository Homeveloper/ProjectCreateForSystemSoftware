package config

// Тесты в этом файле проверяют свойства безопасности, а не функциональность.
// На исходном состоянии продукта (v0.1.0) они не проходят: каждое падение
// соответствует зафиксированному разрыву, который устраняется улучшением.

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// secretValue — значение, которое не должно попадать в вывод и журналы.
const secretValue = "s3cr3t-db-password"

// Разрыв 1: имя файла результата не проверяется, поэтому запись возможна
// за пределы каталога, указанного пользователем.
func TestSaveRejectsPathTraversal(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatalf("подготовка каталога: %v", err)
	}

	names := []string{
		"../escaped.json",
		"sub/../../escaped.json",
	}
	for _, name := range names {
		path, err := Save(root, name, []byte("{}\n"))
		if err == nil {
			t.Errorf("Save(%q): запись выполнена в %s; ожидался отказ, имя выводит за пределы каталога", name, path)
		}
	}
}

// Разрыв 2: результат может содержать секреты, но создаётся с правами 0666.
func TestSaveUsesRestrictiveFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("права доступа POSIX на Windows не применяются")
	}
	dir := t.TempDir()
	path, err := Save(dir, "secrets.json", []byte("{}\n"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		t.Errorf("права файла %04o: результат может содержать секреты и не должен быть доступен другим пользователям", perm)
	}
}

// Разрыв 3: каталог для результата создаётся с правами 0777.
func TestSaveUsesRestrictiveDirectoryPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("права доступа POSIX на Windows не применяются")
	}
	base := t.TempDir()
	nested := filepath.Join(base, "nested")
	if _, err := Save(nested, "out.json", []byte("{}\n")); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(nested)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm&0o027 != 0 {
		t.Errorf("права каталога %04o: ожидались 0750 или строже", perm)
	}
}

// Разрыв 4: файл читается целиком в память без ограничения размера.
func TestLoadRejectsOversizedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "huge.yaml")

	var content bytes.Buffer
	content.WriteString("key: \"")
	content.Write(bytes.Repeat([]byte("x"), 20<<20))
	content.WriteString("\"\n")
	if err := os.WriteFile(path, content.Bytes(), 0o600); err != nil {
		t.Fatalf("подготовка файла: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Error("файл размером 20 МиБ принят целиком; ожидалось ограничение размера входных данных")
	}
}

// Разрыв 5: глубина вложенности не ограничена.
func TestDecodeRejectsExcessiveNesting(t *testing.T) {
	const depth = 10000
	raw := strings.Repeat("{a: ", depth) + "1" + strings.Repeat("}", depth)

	if _, err := Decode([]byte(raw), FormatYAML); err == nil {
		t.Errorf("вложенность глубиной %d принята; ожидалось ограничение глубины", depth)
	}
}

// Разрыв 6: при ошибке разбора в сообщение подставляется всё содержимое файла.
func TestDecodeErrorDoesNotLeakFileContent(t *testing.T) {
	raw := []byte("database:\n  password: " + secretValue + "\n  broken: [unclosed\n")

	_, err := Decode(raw, FormatYAML)
	if err == nil {
		t.Fatal("ожидалась ошибка разбора повреждённой конфигурации")
	}
	if strings.Contains(err.Error(), secretValue) {
		t.Errorf("сообщение об ошибке содержит значение секретного поля:\n%v", err)
	}
}

// Разрыв 7: отчёт о нарушении схемы печатает само значение поля.
func TestValidationErrorDoesNotLeakSecretValue(t *testing.T) {
	schema := &Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"password": {Type: "string", Pattern: "^[A-Z]+$"},
		},
	}
	problems := schema.Validate(Config{"password": secretValue})
	if len(problems) == 0 {
		t.Fatal("ожидалось нарушение схемы")
	}
	joined := strings.Join(problems, "\n")
	if strings.Contains(joined, secretValue) {
		t.Errorf("отчёт о нарушении содержит значение секретного поля:\n%s", joined)
	}
}

// Разрыв 8: список признаков секретных полей неполон.
func TestRedactCoversCommonSecretNames(t *testing.T) {
	cfg := Config{
		"pwd":         "значение-1",
		"credentials": "значение-2",
		"access_key":  "значение-3",
		"private-key": "значение-4",
	}
	redacted := Redact(cfg)
	for key := range cfg {
		if redacted[key] != Mask {
			t.Errorf("поле %q не скрыто, значение осталось в выводе: %v", key, redacted[key])
		}
	}
}
