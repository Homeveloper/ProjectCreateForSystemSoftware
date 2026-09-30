package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	yamlFixture     = "../../testdata/app.yaml"
	jsonFixture     = "../../testdata/app.json"
	overrideFixture = "../../testdata/override.yaml"
	schemaFixture   = "../../testdata/schema.json"
	invalidFixture  = "../../testdata/invalid.yaml"
)

func TestDetectFormat(t *testing.T) {
	cases := map[string]Format{
		"app.yaml":        FormatYAML,
		"app.YML":         FormatYAML,
		"config/app.json": FormatJSON,
	}
	for name, want := range cases {
		got, err := DetectFormat(name)
		if err != nil {
			t.Fatalf("DetectFormat(%q): неожиданная ошибка %v", name, err)
		}
		if got != want {
			t.Errorf("DetectFormat(%q) = %q, ожидалось %q", name, got, want)
		}
	}
	if _, err := DetectFormat("app.toml"); err == nil {
		t.Error("DetectFormat(app.toml): ожидалась ошибка для неизвестного расширения")
	}
}

func TestLoadYAML(t *testing.T) {
	cfg, err := Load(yamlFixture)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	service, ok := cfg["service"].(map[string]interface{})
	if !ok {
		t.Fatalf("service: ожидался объект, получено %T", cfg["service"])
	}
	if service["name"] != "billing-api" {
		t.Errorf("service.name = %v, ожидалось billing-api", service["name"])
	}
	if service["port"] != 8080 {
		t.Errorf("service.port = %v (%T), ожидалось 8080", service["port"], service["port"])
	}
}

func TestLoadJSON(t *testing.T) {
	cfg, err := Load(jsonFixture)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if _, ok := cfg["database"].(map[string]interface{}); !ok {
		t.Fatalf("database: ожидался объект, получено %T", cfg["database"])
	}
}

func TestLoadRejectsNonObjectRoot(t *testing.T) {
	if _, err := Decode([]byte("- a\n- b\n"), FormatYAML); err == nil {
		t.Error("ожидалась ошибка: корень конфигурации не объект")
	}
}

func TestEncodeRoundTrip(t *testing.T) {
	cfg, err := Load(yamlFixture)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	first, err := Encode(cfg, FormatJSON)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	reparsed, err := Decode(first, FormatJSON)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	second, err := Encode(reparsed, FormatJSON)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(first) != string(second) {
		t.Errorf("преобразование не устойчиво:\nпервое:\n%s\nвторое:\n%s", first, second)
	}
}

func TestMerge(t *testing.T) {
	base, err := Load(yamlFixture)
	if err != nil {
		t.Fatalf("Load base: %v", err)
	}
	override, err := Load(overrideFixture)
	if err != nil {
		t.Fatalf("Load override: %v", err)
	}
	merged := Merge(base, override)
	service := merged["service"].(map[string]interface{})
	if service["port"] != 9090 {
		t.Errorf("service.port = %v, ожидалось 9090 из override", service["port"])
	}
	if service["name"] != "billing-api" {
		t.Errorf("service.name = %v, ожидалось сохранение billing-api из base", service["name"])
	}
	database := merged["database"].(map[string]interface{})
	if database["port"] != 5432 {
		t.Errorf("database.port = %v, ожидалось сохранение 5432", database["port"])
	}
	baseService := base["service"].(map[string]interface{})
	if baseService["port"] != 8080 {
		t.Errorf("Merge изменил исходную конфигурацию: service.port = %v", baseService["port"])
	}
}

func TestRedact(t *testing.T) {
	cfg, err := Load(yamlFixture)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	redacted := Redact(cfg)
	database := redacted["database"].(map[string]interface{})
	if database["password"] != Mask {
		t.Errorf("database.password = %v, ожидалась маска", database["password"])
	}
	if database["user"] != "billing" {
		t.Errorf("database.user = %v, несекретное поле изменено", database["user"])
	}
	auth := redacted["auth"].(map[string]interface{})
	if auth["api_token"] != Mask {
		t.Errorf("auth.api_token = %v, ожидалась маска", auth["api_token"])
	}
}

func TestSchemaAcceptsValidConfig(t *testing.T) {
	schema, err := LoadSchema(schemaFixture)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}
	cfg, err := Load(yamlFixture)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := schema.Validate(cfg); len(problems) != 0 {
		t.Errorf("ожидалось отсутствие нарушений, получено: %v", problems)
	}
}

func TestSchemaReportsProblems(t *testing.T) {
	schema, err := LoadSchema(schemaFixture)
	if err != nil {
		t.Fatalf("LoadSchema: %v", err)
	}
	cfg, err := Load(invalidFixture)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	problems := schema.Validate(cfg)
	if len(problems) != 4 {
		t.Fatalf("ожидалось 4 нарушения, получено %d: %v", len(problems), problems)
	}
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"database", "name", "port", "mode"} {
		if !strings.Contains(joined, want) {
			t.Errorf("в отчёте нет упоминания %q:\n%s", want, joined)
		}
	}
}

func TestSaveWritesFile(t *testing.T) {
	dir := t.TempDir()
	path, err := Save(dir, "out.json", []byte("{}\n"))
	if err != nil {
		t.Fatalf("Save: %v", err)
	}
	if want := filepath.Join(dir, "out.json"); path != want {
		t.Errorf("path = %q, ожидалось %q", path, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "{}\n" {
		t.Errorf("содержимое = %q", data)
	}
}
