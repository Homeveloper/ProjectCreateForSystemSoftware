package cli

import "errors"

// Коды завершения. Разные причины отказа различаются по коду,
// чтобы вызывающий сценарий мог реагировать на них по-разному,
// не разбирая текст сообщения.
const (
	// ExitOK — работа завершена успешно.
	ExitOK = 0
	// ExitUsage — ошибка в аргументах командной строки.
	ExitUsage = 1
	// ExitInput — входной файл не прочитан или не разобран,
	// в том числе из-за превышения ограничений.
	ExitInput = 2
	// ExitSchema — конфигурация разобрана, но нарушает схему.
	ExitSchema = 3
	// ExitOutput — результат не записан.
	ExitOutput = 4
)

// exitError связывает ошибку с кодом завершения.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

func withCode(code int, err error) error {
	if err == nil {
		return nil
	}
	var existing *exitError
	if errors.As(err, &existing) {
		return err
	}
	return &exitError{code: code, err: err}
}

// ExitCodeFor возвращает код завершения, соответствующий ошибке.
func ExitCodeFor(err error) int {
	if err == nil {
		return ExitOK
	}
	var coded *exitError
	if errors.As(err, &coded) {
		return coded.code
	}
	return ExitUsage
}
