package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Config — разобранная конфигурация. Корень всегда объект.
type Config map[string]interface{}

// Load читает файл конфигурации с ограничениями по умолчанию.
func Load(path string) (Config, error) {
	return LoadWithLimits(path, DefaultLimits)
}

// LoadWithLimits читает файл конфигурации, не выходя за заданные ограничения.
func LoadWithLimits(path string, limits Limits) (Config, error) {
	limits = limits.resolve()

	format, err := DetectFormat(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("открытие %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("чтение сведений о %s: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s не является обычным файлом", path)
	}
	if err := limits.checkSize(info.Size()); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	// Читатель ограничен дополнительно: файл мог вырасти между
	// проверкой размера и чтением.
	raw, err := io.ReadAll(io.LimitReader(file, limits.MaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("чтение %s: %w", path, err)
	}
	if err := limits.checkSize(int64(len(raw))); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	cfg, err := DecodeWithLimits(raw, format, limits)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

// Decode разбирает содержимое конфигурации с ограничениями по умолчанию.
func Decode(raw []byte, format Format) (Config, error) {
	return DecodeWithLimits(raw, format, DefaultLimits)
}

// DecodeWithLimits разбирает содержимое конфигурации в указанном формате.
//
// Сообщения об ошибках не содержат разбираемых данных: конфигурации
// хранят пароли и маркеры доступа, которые не должны попадать
// в журналы и вывод программы.
func DecodeWithLimits(raw []byte, format Format, limits Limits) (Config, error) {
	limits = limits.resolve()

	if err := limits.checkSize(int64(len(raw))); err != nil {
		return nil, err
	}
	if err := limits.checkFlowDepth(raw); err != nil {
		return nil, err
	}

	var value interface{}
	switch format {
	case FormatYAML:
		if err := yaml.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("разбор YAML (%s): %w", describeContent(raw), err)
		}
	case FormatJSON:
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, fmt.Errorf("разбор JSON (%s): %w", describeContent(raw), err)
		}
	default:
		return nil, fmt.Errorf("неподдерживаемый формат %q", format)
	}

	if err := limits.checkValueDepth(value, 0); err != nil {
		return nil, err
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
		data, err := yaml.Marshal(map[string]interface{}(cfg))
		if err != nil {
			return nil, fmt.Errorf("сериализация YAML: %w", err)
		}
		return data, nil
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

// normalize приводит ключи отображений к строковому виду.
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
