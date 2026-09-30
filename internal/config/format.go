package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Format — поддерживаемый формат конфигурации.
type Format string

const (
	FormatYAML Format = "yaml"
	FormatJSON Format = "json"
)

// DetectFormat определяет формат по расширению имени файла.
func DetectFormat(name string) (Format, error) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".yaml", ".yml":
		return FormatYAML, nil
	case ".json":
		return FormatJSON, nil
	default:
		return "", fmt.Errorf("не удалось определить формат по имени %q (ожидается .yaml, .yml или .json)", name)
	}
}
