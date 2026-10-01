package config

import (
	"fmt"
	"sort"
)

// Summary — краткое описание конфигурации для режима просмотра.
// Описание содержит только имена и количества, но не значения:
// просмотр не должен раскрывать содержимое конфигурации.
type Summary struct {
	// TopLevelKeys — число полей верхнего уровня.
	TopLevelKeys int
	// Fields — общее число полей объектов на всех уровнях.
	Fields int
	// Leaves — число конечных значений.
	Leaves int
	// MaxDepth — наибольшая глубина вложенности.
	MaxDepth int
	// SecretFields — пути полей, опознанных как секретные.
	SecretFields []string
}

// Summarize составляет краткое описание конфигурации.
func Summarize(cfg Config) Summary {
	summary := Summary{TopLevelKeys: len(cfg)}
	summary.walk(map[string]interface{}(cfg), "$", 0)
	sort.Strings(summary.SecretFields)
	return summary
}

func (s *Summary) walk(value interface{}, at string, depth int) {
	if depth > s.MaxDepth {
		s.MaxDepth = depth
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, item := range typed {
			s.Fields++
			path := at + "." + key
			if IsSecretKey(key) {
				s.SecretFields = append(s.SecretFields, path)
				// Значение секретного поля не обходится: его структура
				// описания не добавляет, а путь уже записан.
				continue
			}
			s.walk(item, path, depth+1)
		}
	case []interface{}:
		for i, item := range typed {
			s.walk(item, fmt.Sprintf("%s[%d]", at, i), depth+1)
		}
	default:
		s.Leaves++
	}
}
