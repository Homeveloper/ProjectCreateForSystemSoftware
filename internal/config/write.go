package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// filePerm — права на файл результата. Результат может содержать
	// секреты, поэтому доступ имеет только владелец.
	filePerm os.FileMode = 0o600
	// dirPerm — права на создаваемый каталог.
	dirPerm os.FileMode = 0o700
)

// Save записывает данные в файл name внутри каталога outDir
// и возвращает итоговый путь.
//
// Имя результата приходит из аргументов командной строки и считается
// недоверенным: запись за пределы outDir отклоняется, как и запись
// через символьную ссылку.
func Save(outDir, name string, data []byte) (string, error) {
	if name == "" {
		return "", fmt.Errorf("имя файла результата не задано")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("имя результата %q задано абсолютным путём; ожидается имя внутри каталога %s", name, outDir)
	}

	root, err := filepath.Abs(outDir)
	if err != nil {
		return "", fmt.Errorf("определение каталога %s: %w", outDir, err)
	}

	path := filepath.Join(root, name)
	if err := ensureInside(root, path); err != nil {
		return "", fmt.Errorf("имя результата %q отклонено: %w", name, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return "", fmt.Errorf("создание каталога для %s: %w", path, err)
	}
	if err := refuseSymlink(path); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, filePerm); err != nil {
		return "", fmt.Errorf("запись %s: %w", path, err)
	}
	return path, nil
}

// ensureInside проверяет, что path не выходит за пределы root.
func ensureInside(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return fmt.Errorf("путь не удалось соотнести с каталогом %s: %w", root, err)
	}
	if relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("путь выводит за пределы каталога %s", root)
	}
	return nil
}

// refuseSymlink отклоняет запись, если по указанному пути уже находится
// символьная ссылка: иначе запись ушла бы в произвольное место.
func refuseSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("проверка %s: %w", path, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s является символьной ссылкой; запись отклонена", path)
	}
	return nil
}
