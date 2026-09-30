package config

import "strings"

// Mask подставляется вместо значений полей, признанных секретными.
const Mask = "***REDACTED***"

// secretHints — признаки секретного поля. Сравнение выполняется
// с нормализованным именем, поэтому разделители в них не нужны:
// "access_key", "access-key" и "AccessKey" опознаются одинаково.
var secretHints = []string{
	"password",
	"passwd",
	"pwd",
	"passphrase",
	"secret",
	"token",
	"apikey",
	"privatekey",
	"accesskey",
	"credential",
	"signature",
}

// separatorRemover приводит имя поля к виду без разделителей.
var separatorRemover = strings.NewReplacer("-", "", "_", "", ".", "", " ", "")

// IsSecretKey сообщает, считается ли имя поля секретным.
//
// Проверка намеренно широкая: ложно скрытое значение безвредно,
// а раскрытый пароль — нет.
func IsSecretKey(key string) bool {
	normalized := separatorRemover.Replace(strings.ToLower(key))
	for _, hint := range secretHints {
		if strings.Contains(normalized, hint) {
			return true
		}
	}
	return false
}

// Redact возвращает копию конфигурации со скрытыми значениями секретных полей.
func Redact(cfg Config) Config {
	out, _ := redactValue(map[string]interface{}(cfg)).(map[string]interface{})
	return Config(out)
}

func redactValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			if IsSecretKey(key) {
				out[key] = Mask
				continue
			}
			out[key] = redactValue(item)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(typed))
		for i, item := range typed {
			out[i] = redactValue(item)
		}
		return out
	default:
		return value
	}
}
