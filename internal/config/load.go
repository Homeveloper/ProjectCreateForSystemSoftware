package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// Config — разобранная конфигурация. Корень всегда объект.
type Config map[string]interface{}

// Load читает файл конфигурации и разбирает его.
func Load(path string) (Config, error) {
	format, err := DetectFormat(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение %s: %w", path, err)
	}
	cfg, err := Decode(raw, format)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Decode разбирает содержимое конфигурации в указанном формате.
func Decode(raw []byte, format Format) (Config, error) {
	var value interface{}
	switch format {
	case FormatYAML:
		if err := yaml.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("разбор YAML: %w; содержимое: %s", err, string(raw))
		}
	case FormatJSON:
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("разбор JSON: %w; содержимое: %s", err, string(raw))
		}
	default:
		return nil, fmt.Errorf("неподдерживаемый формат %q", format)
	}
	normalized, ok := normalize(value).(map[string]interface{})
	if !ok {
		return nil, errors.New("корень конфигурации должен быть объектом")
	}
	return Config(normalized), nil
}

// Encode сериализует конфигурацию в указанный формат.
func Encode(cfg Config, format Format) ([]byte, error) {
	switch format {
	case FormatYAML:
		return yaml.Marshal(map[string]interface{}(cfg))
	case FormatJSON:
		data, err := json.MarshalIndent(map[string]interface{}(cfg), "", "  ")
		if err != nil {
			return nil, fmt.Errorf("сериализация JSON: %w", err)
		}
		return append(data, '\n'), nil
	default:
		return nil, fmt.Errorf("неподдерживаемый формат %q", format)
	}
}

// normalize приводит структуры yaml.v2 (map[interface{}]interface{})
// к виду map[string]interface{}, пригодному для дальнейшей работы.
func normalize(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			out[fmt.Sprint(key)] = normalize(item)
		}
		return out
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			out[key] = normalize(item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(typed))
		for i, item := range typed {
			out[i] = normalize(item)
		}
		return out
	default:
		return value
	}
}
