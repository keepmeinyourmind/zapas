package domain

import "errors"

var (
	ErrNotFound       = errors.New("Не найдено")
	ErrDecisionClosed = errors.New("Вопрос уже закрыт")
)

// ValidationError — некорректные данные от клиента API (HTTP 400).
type ValidationError string

func (e ValidationError) Error() string { return string(e) }
