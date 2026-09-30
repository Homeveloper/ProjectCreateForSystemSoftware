package config

import "strings"

// Mask подставляется вместо значений полей, признанных секретными.
const Mask = "***REDACTED***"

// secretHints — подстроки в именах полей, по которым значение считается секретом.
var secretHints = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"apikey",
	"api_key",
	"private_key",
}

// IsSecretKey сообщает, считается ли имя поля секретным.
func IsSecretKey(key string) bool {
	lower := strings.ToLower(key)
	for _, hint := range secretHints {
		if strings.Contains(lower, hint) {
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
