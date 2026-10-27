package domain

import "testing"

func TestMoneyString(t *testing.T) {
	tests := []struct {
		name string
		in   Money
		want string
	}{
		{"ноль", 0, "0 ₽"},
		{"целые рубли", FromRubles(3000), "3 000 ₽"},
		{"с копейками", 125050, "1 250,50 ₽"},
		{"миллион", FromRubles(1_234_567), "1 234 567 ₽"},
		{"отрицательная", -FromRubles(1400), "-1 400 ₽"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.String(); got != tt.want {
				t.Errorf("Money(%d).String() = %q, want %q", int64(tt.in), got, tt.want)
			}
		})
	}
}

func TestMoneyPercent(t *testing.T) {
	// 10% от зарплаты Ани.
	if got := FromRubles(30000).Share(10); got != FromRubles(3000) {
		t.Errorf("10%% от 30 000 ₽ = %v, want 3 000 ₽", got)
	}
	// Округление вниз до копейки: 10% от 0,15 ₽ = 0,01 ₽ (не 0,015).
	if got := Money(15).Share(10); got != 1 {
		t.Errorf("10%% от 15 коп = %d коп, want 1", int64(got))
	}
}

func TestRuleValidate(t *testing.T) {
	valid := SavingsRule{ClientID: "anya", GoalAmount: FromRubles(36000), TermMonths: 12, SavePercent: 10, MinTransfer: DefaultMinTransfer}

	if err := valid.Validate(); err != nil {
		t.Fatalf("корректное правило отклонено: %v", err)
	}

	broken := []struct {
		name   string
		mutate func(r *SavingsRule)
	}{
		{"без клиента", func(r *SavingsRule) { r.ClientID = "" }},
		{"нулевая цель", func(r *SavingsRule) { r.GoalAmount = 0 }},
		{"срок 7 месяцев", func(r *SavingsRule) { r.TermMonths = 7 }},
		{"процент 0", func(r *SavingsRule) { r.SavePercent = 0 }},
		{"процент 60", func(r *SavingsRule) { r.SavePercent = 60 }},
		{"отрицательный порог", func(r *SavingsRule) { r.MinIncome = -1 }},
		{"минимальный перевод 1 копейка", func(r *SavingsRule) { r.MinTransfer = 1 }},
	}
	for _, tt := range broken {
		t.Run(tt.name, func(t *testing.T) {
			r := valid // копия структуры, исходник не меняется
			tt.mutate(&r)
			if err := r.Validate(); err == nil {
				t.Errorf("ожидали ошибку для случая %q", tt.name)
			}
		})
	}
}

func TestIncomeKindIsIncome(t *testing.T) {
	if IncomeNone.IsIncome() {
		t.Error("not_income не должен считаться доходом")
	}
	if !IncomeSalary.IsIncome() {
		t.Error("зарплата должна считаться доходом")
	}
}
