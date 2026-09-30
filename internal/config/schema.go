package config

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Schema — поддерживаемое подмножество JSON Schema.
type Schema struct {
	Type       string             `json:"type"`
	Required   []string           `json:"required"`
	Properties map[string]*Schema `json:"properties"`
	Items      *Schema            `json:"items"`
	Enum       []interface{}      `json:"enum"`
	Pattern    string             `json:"pattern"`
	Minimum    *float64           `json:"minimum"`
	Maximum    *float64           `json:"maximum"`
}

// LoadSchema читает схему из файла JSON.
func LoadSchema(path string) (*Schema, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("чтение схемы %s: %w", path, err)
	}
	var schema Schema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("разбор схемы %s: %w", path, err)
	}
	return &schema, nil
}

// Validate проверяет конфигурацию по схеме и возвращает список нарушений.
// Пустой список означает, что конфигурация корректна.
func (s *Schema) Validate(cfg Config) []string {
	var problems []string
	s.check(map[string]interface{}(cfg), "$", &problems)
	sort.Strings(problems)
	return problems
}

func (s *Schema) check(value interface{}, at string, problems *[]string) {
	if s == nil {
		return
	}
	if s.Type != "" && !matchesType(value, s.Type) {
		*problems = append(*problems,
			fmt.Sprintf("%s: ожидался тип %s, получен %s", at, s.Type, typeName(value)))
		return
	}
	switch s.Type {
	case "object":
		object, _ := value.(map[string]interface{})
		for _, name := range s.Required {
			if _, ok := object[name]; !ok {
				*problems = append(*problems,
					fmt.Sprintf("%s: отсутствует обязательное поле %q", at, name))
			}
		}
		for name, sub := range s.Properties {
			if item, ok := object[name]; ok {
				sub.check(item, at+"."+name, problems)
			}
		}
	case "array":
		items, _ := value.([]interface{})
		for i, item := range items {
			s.Items.check(item, fmt.Sprintf("%s[%d]", at, i), problems)
		}
	case "string":
		text, _ := value.(string)
		if s.Pattern != "" {
			pattern, err := regexp.Compile(s.Pattern)
			if err != nil {
				*problems = append(*problems,
					fmt.Sprintf("%s: некорректный шаблон в схеме: %v", at, err))
			} else if !pattern.MatchString(text) {
				*problems = append(*problems,
					fmt.Sprintf("%s: значение %s не соответствует шаблону %q", at, presentValue(at, text), s.Pattern))
			}
		}
	case "number", "integer":
		if number, ok := toFloat(value); ok {
			if s.Minimum != nil && number < *s.Minimum {
				*problems = append(*problems,
					fmt.Sprintf("%s: значение %s меньше минимального %v", at, presentValue(at, number), *s.Minimum))
			}
			if s.Maximum != nil && number > *s.Maximum {
				*problems = append(*problems,
					fmt.Sprintf("%s: значение %s больше максимального %v", at, presentValue(at, number), *s.Maximum))
			}
		}
	}
	if len(s.Enum) > 0 && !inEnum(value, s.Enum) {
		*problems = append(*problems,
			fmt.Sprintf("%s: значение %s не входит в допустимый набор", at, presentValue(at, value)))
	}
}

func matchesType(value interface{}, want string) bool {
	switch want {
	case "object":
		_, ok := value.(map[string]interface{})
		return ok
	case "array":
		_, ok := value.([]interface{})
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "integer":
		switch number := value.(type) {
		case int, int64, uint64:
			return true
		case float64:
			return number == math.Trunc(number)
		default:
			return false
		}
	case "number":
		_, ok := toFloat(value)
		return ok
	case "null":
		return value == nil
	default:
		return true
	}
}

func typeName(value interface{}) string {
	switch value.(type) {
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	case string:
		return "string"
	case bool:
		return "boolean"
	case int, int64, uint64, float64:
		return "number"
	case nil:
		return "null"
	default:
		return fmt.Sprintf("%T", value)
	}
}

func toFloat(value interface{}) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case uint64:
		return float64(number), true
	case float64:
		return number, true
	default:
		return 0, false
	}
}

func inEnum(value interface{}, allowed []interface{}) bool {
	for _, candidate := range allowed {
		if fmt.Sprint(candidate) == fmt.Sprint(value) {
			return true
		}
	}
	return false
}

// fieldName выделяет имя поля из пути вида "$.database.password".
func fieldName(at string) string {
	name := at
	if index := strings.LastIndex(name, "["); index >= 0 {
		name = name[:index]
	}
	if index := strings.LastIndex(name, "."); index >= 0 {
		name = name[index+1:]
	}
	return name
}

// presentValue готовит значение к выводу в отчёте о нарушениях.
//
// Отчёт попадает в журналы и в вывод сборочного конвейера, поэтому
// значения полей, опознанных как секретные, заменяются маской.
func presentValue(at string, value interface{}) string {
	if IsSecretKey(fieldName(at)) {
		return Mask
	}
	if text, ok := value.(string); ok {
		return fmt.Sprintf("%q", text)
	}
	return fmt.Sprintf("%v", value)
}
