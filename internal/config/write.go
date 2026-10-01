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

// SaveOptions управляют записью результата.
type SaveOptions struct {
	// Overwrite разрешает замену существующего файла.
	// По умолчанию запись поверх существующего файла отклоняется:
	// утилита не должна уничтожать данные неожиданно для пользователя.
	Overwrite bool
}

// ErrOutputExists возвращается, когда файл результата уже существует,
// а замена не разрешена.
type ErrOutputExists struct {
	Path string
}

func (e *ErrOutputExists) Error() string {
	return fmt.Sprintf("файл %s уже существует; для замены укажите --force", e.Path)
}

// Save записывает данные в файл name внутри каталога outDir,
// не заменяя существующий файл.
func Save(outDir, name string, data []byte) (string, error) {
	return SaveWithOptions(outDir, name, data, SaveOptions{})
}

// SaveWithOptions записывает данные в файл name внутри каталога outDir
// и возвращает итоговый путь.
//
// Имя результата приходит из аргументов командной строки и считается
// недоверенным: запись за пределы outDir отклоняется, как и запись
// через символьную ссылку. Запись выполняется через временный файл
// с последующим переименованием, поэтому прерывание не оставляет
// частично записанного результата.
func SaveWithOptions(outDir, name string, data []byte, opts SaveOptions) (string, error) {
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
	if err := checkTarget(path, opts.Overwrite); err != nil {
		return "", err
	}
	if err := writeAtomic(path, data); err != nil {
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

// checkTarget проверяет состояние пути назначения до записи.
func checkTarget(path string, overwrite bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("проверка %s: %w", path, err)
	}
	// Символьная ссылка отклоняется всегда: запись ушла бы
	// в произвольное место за пределами каталога результата.
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s является символьной ссылкой; запись отклонена", path)
	}
	if info.IsDir() {
		return fmt.Errorf("%s является каталогом; запись отклонена", path)
	}
	if !overwrite {
		return &ErrOutputExists{Path: path}
	}
	return nil
}

// writeAtomic записывает данные во временный файл рядом с целевым
// и переименовывает его. Прерывание на любом шаге оставляет либо
// прежний файл, либо отсутствие файла, но не обрезанный результат.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".cfgtool-*.tmp")
	if err != nil {
		return fmt.Errorf("создание временного файла: %w", err)
	}
	tempName := temp.Name()

	// Временный файл удаляется, если переименование не состоялось.
	committed := false
	defer func() {
		if !committed {
			_ = os.Remove(tempName)
		}
	}()

	if err := temp.Chmod(filePerm); err != nil {
		_ = temp.Close()
		return fmt.Errorf("установка прав на временный файл: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("запись во временный файл: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("сброс временного файла на диск: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("закрытие временного файла: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("перемещение временного файла: %w", err)
	}
	committed = true
	return nil
}
