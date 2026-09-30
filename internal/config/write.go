package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Save записывает данные в файл name внутри каталога outDir
// и возвращает итоговый путь.
func Save(outDir, name string, data []byte) (string, error) {
	path := filepath.Join(outDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return "", fmt.Errorf("создание каталога для %s: %w", path, err)
	}
	if err := os.WriteFile(path, data, 0666); err != nil {
		return "", fmt.Errorf("запись %s: %w", path, err)
	}
	return path, nil
}
