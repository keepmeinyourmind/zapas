package domain

import "errors"

var (
	ErrNotFound       = errors.New("Не найдено")
	ErrDecisionClosed = errors.New("Вопрос уже закрыт")
)

// ValidationError — некорректные данные от клиента API (HTTP 400).
type ValidationError string

func (e ValidationError) Error() string { return string(e) }

// ErrOnboardingRequired — правило копилки нельзя создать без одобренной анкеты (HTTP 409).
var ErrOnboardingRequired = errors.New("Сначала заполните анкету: без неё копилку подключить нельзя")

// ErrAccountArrested — счёт арестован, переводы и ответы на вопросы невозможны (HTTP 409).
var ErrAccountArrested = errors.New("Счёт арестован, переводы невозможны")

// ErrNotEligible — по итогам анкеты копилка клиенту недоступна (HTTP 409).
var ErrNotEligible = errors.New("По итогам анкеты копилка сейчас недоступна")
