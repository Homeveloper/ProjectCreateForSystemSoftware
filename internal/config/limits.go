package config

import (
	"fmt"
	"unicode/utf8"
)

// Limits ограничивают объём и сложность разбираемых данных.
// Недоверенный файл конфигурации не должен приводить к исчерпанию
// памяти или стека, поэтому каждая граница задана явно.
type Limits struct {
	// MaxBytes — предельный размер входных данных в байтах.
	MaxBytes int64
	// MaxDepth — предельная глубина вложенности структур.
	MaxDepth int
}

// DefaultLimits применяются, если вызывающая сторона не задала свои.
var DefaultLimits = Limits{
	MaxBytes: 10 << 20, // 10 МиБ
	MaxDepth: 64,
}

// checkSize проверяет объём входных данных.
func (l Limits) checkSize(size int64) error {
	if size > l.MaxBytes {
		return fmt.Errorf("размер входных данных %d байт превышает допустимый предел %d байт",
			size, l.MaxBytes)
	}
	return nil
}

// checkFlowDepth оценивает глубину вложенности до разбора.
//
// Проверка нужна именно до разбора: конструкция вида "{a: {a: ...}}"
// глубиной в десятки тысяч уровней исчерпывает стек ещё внутри
// синтаксического анализатора, то есть до того, как структуру можно
// было бы обойти и измерить.
//
// Учитываются только указатели потокового стиля, находящиеся вне строк
// в кавычках, поэтому скобки внутри значений не считаются вложенностью.
func (l Limits) checkFlowDepth(raw []byte) error {
	const (
		plain = iota
		inSingleQuotes
		inDoubleQuotes
		inComment
	)

	state := plain
	depth := 0

	for i := 0; i < len(raw); i++ {
		char := raw[i]
		switch state {
		case inSingleQuotes:
			if char == '\'' {
				state = plain
			}
		case inDoubleQuotes:
			switch char {
			case 0x5c: // обратная косая черта
				i++ // экранированный символ не завершает строку
			case '"':
				state = plain
			}
		case inComment:
			if char == '\n' {
				state = plain
			}
		default:
			switch char {
			case '\'':
				state = inSingleQuotes
			case '"':
				state = inDoubleQuotes
			case '#':
				state = inComment
			case '{', '[':
				depth++
				if depth > l.MaxDepth {
					return fmt.Errorf("глубина вложенности превышает допустимый предел %d", l.MaxDepth)
				}
			case '}', ']':
				if depth > 0 {
					depth--
				}
			}
		}
	}
	return nil
}

// checkValueDepth проверяет глубину уже разобранной структуры.
// Дополняет checkFlowDepth: блочный стиль YAML скобок не использует.
func (l Limits) checkValueDepth(value interface{}, depth int) error {
	if depth > l.MaxDepth {
		return fmt.Errorf("глубина вложенности превышает допустимый предел %d", l.MaxDepth)
	}
	switch typed := value.(type) {
	case map[string]interface{}:
		for _, item := range typed {
			if err := l.checkValueDepth(item, depth+1); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, item := range typed {
			if err := l.checkValueDepth(item, depth+1); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolve возвращает действующие ограничения, подставляя значения
// по умолчанию вместо незаданных.
func (l Limits) resolve() Limits {
	if l.MaxBytes <= 0 {
		l.MaxBytes = DefaultLimits.MaxBytes
	}
	if l.MaxDepth <= 0 {
		l.MaxDepth = DefaultLimits.MaxDepth
	}
	return l
}

// describeContent кратко описывает входные данные для сообщения об ошибке,
// не раскрывая их содержимого: конфигурации содержат пароли и маркеры доступа.
func describeContent(raw []byte) string {
	return fmt.Sprintf("%d байт, %d символов", len(raw), utf8.RuneCount(raw))
}
