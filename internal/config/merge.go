package config

// Merge накладывает override на base и возвращает новую конфигурацию.
// Вложенные объекты объединяются рекурсивно, остальные значения заменяются.
func Merge(base, override Config) Config {
	out := make(Config, len(base))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range override {
		if existing, ok := out[key]; ok {
			baseMap, baseOK := existing.(map[string]interface{})
			overrideMap, overrideOK := value.(map[string]interface{})
			if baseOK && overrideOK {
				out[key] = map[string]interface{}(Merge(Config(baseMap), Config(overrideMap)))
				continue
			}
		}
		out[key] = value
	}
	return out
}
