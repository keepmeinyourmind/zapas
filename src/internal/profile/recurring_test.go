package profile

import (
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

func on(m time.Month, d int) time.Time { return time.Date(2026, m, d, 10, 0, 0, 0, time.UTC) }

func spend(to string, rubles int64, at time.Time) domain.Transaction {
	return domain.Transaction{
		Amount: domain.FromRubles(rubles), Type: domain.TxKindOutgoing,
		Counterparty: to, CounterpartyKind: domain.CounterpartyLegal, OccurredAt: at,
	}
}

// anyaSpending — списания Ани за полгода: рассрочка 25-го, связь 10-го, подписка 3-го,
// продукты каждую неделю, разовые и нерегулярные траты.
func anyaSpending() []domain.Transaction {
	var h []domain.Transaction
	for m := time.May; m <= time.October; m++ {
		if m <= time.September { // за октябрь рассрочка ещё не списалась
			h = append(h, spend("Рассрочка за ноутбук", 6000, on(m, 25)))
		}
		h = append(h, spend("МТС", 500, on(m, 10)))
		h = append(h, spend("Яндекс Плюс", 499, on(m, 3)))
	}
	for d := 1; d <= 28; d += 7 { // продукты раз в неделю — не «раз в месяц»
		h = append(h, spend("Пятёрочка", 1200, on(time.October, d)))
	}
	h = append(h,
		spend("Кино", 600, on(time.July, 12)), spend("Кино", 600, on(time.September, 12)), // раз в 2 месяца
		spend("Такси", 300, on(time.August, 5)), spend("Такси", 900, on(time.September, 5)), // сумма скачет
		spend("Спортзал", 2000, on(time.May, 15)), spend("Спортзал", 2000, on(time.June, 15)), // бросила в июне
	)
	return h
}

func TestDetectRecurringOnAnyaHistory(t *testing.T) {
	got := DetectRecurring(anyaSpending(), on(time.October, 20))

	want := []domain.RecurringPayment{ // по алфавиту: результат детерминированный
		{Counterparty: "МТС", Amount: domain.FromRubles(500), DayOfMonth: 10, LastAt: on(time.October, 10)},
		{Counterparty: "Рассрочка за ноутбук", Amount: domain.FromRubles(6000), DayOfMonth: 25, LastAt: on(time.September, 25)},
		{Counterparty: "Яндекс Плюс", Amount: domain.FromRubles(499), DayOfMonth: 3, LastAt: on(time.October, 3)},
	}
	if len(got) != len(want) {
		t.Fatalf("найдено %d платежей, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("платёж %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestDetectRecurring(t *testing.T) {
	now := on(time.October, 20)
	tests := []struct {
		name    string
		history []domain.Transaction
		want    int // сколько регулярных платежей
	}{
		{"нет истории", nil, 0},
		{"одно списание — ещё не регулярное", []domain.Transaction{spend("МТС", 500, on(time.October, 10))}, 0},
		{"два раза с интервалом месяц — регулярное", []domain.Transaction{
			spend("МТС", 500, on(time.September, 10)), spend("МТС", 500, on(time.October, 10))}, 1},
		{"сумма в пределах ±10% — регулярное", []domain.Transaction{
			spend("МТС", 500, on(time.September, 10)), spend("МТС", 540, on(time.October, 10))}, 1},
		{"сумма скачет больше 10% — не регулярное", []domain.Transaction{
			spend("Такси", 300, on(time.September, 10)), spend("Такси", 900, on(time.October, 10))}, 0},
		{"интервал 2 месяца — не регулярное", []domain.Transaction{
			spend("Кино", 600, on(time.August, 10)), spend("Кино", 600, on(time.October, 10))}, 0},
		{"перестала платить — не регулярное", []domain.Transaction{
			spend("Спортзал", 2000, on(time.May, 15)), spend("Спортзал", 2000, on(time.June, 15))}, 0},
		{"входящие не считаются", []domain.Transaction{
			{Amount: domain.FromRubles(500), Type: domain.TxKindIncoming, Counterparty: "МТС", OccurredAt: on(time.September, 10)},
			{Amount: domain.FromRubles(500), Type: domain.TxKindIncoming, Counterparty: "МТС", OccurredAt: on(time.October, 10)}}, 0},
		{"история не по порядку", []domain.Transaction{
			spend("МТС", 500, on(time.October, 10)), spend("МТС", 500, on(time.September, 10))}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectRecurring(tt.history, now); len(got) != tt.want {
				t.Errorf("найдено %d, want %d: %+v", len(got), tt.want, got)
			}
		})
	}
}

func TestDailySpending(t *testing.T) {
	h := anyaSpending()
	now := on(time.October, 20)
	recurring := DetectRecurring(h, now)
	// За 30 дней до 20 октября: продукты 1, 8 и 15 октября по 1 200 ₽ (22-го — ещё не было) = 3 600 ₽.
	// Регулярные (МТС, подписка) и траты раньше 20 сентября не считаются. 3 600 / 30 = 120 ₽ в день.
	if got := DailySpending(h, recurring, now); got != domain.FromRubles(120) {
		t.Errorf("DailySpending = %v, want 120 ₽", got)
	}
}
