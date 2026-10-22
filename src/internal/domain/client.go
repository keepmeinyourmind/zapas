package domain

// Client — клиент и его балансы.
type Client struct {
	ID             string
	Name           string
	CardBalance    Money // собственные деньги на карте
	CreditLimit    Money // доступный кредит; в расчёте безопасной суммы НЕ участвует
	SavingsBalance Money // баланс копилки
}

// ClassifiedTransaction — операция вместе с меткой классификатора.
// Вложенная структура Transaction в JSON «разворачивается» в те же поля.
type ClassifiedTransaction struct {
	Transaction
	IncomeKind IncomeKind `json:"income_kind"`
}
