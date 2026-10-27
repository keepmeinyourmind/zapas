package engine

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/keepmeinyourmind/zapas/src/internal/domain"
)

var cfg = Config{DropPct: 15, RaisePct: 15, Buffer: 0}

// anya — обычный случай из демо: пришла зарплата 30 000 ₽, всё как всегда.
// Каждый тест берёт этот вход и меняет одно-два поля.
func anya() Input {
	return Input{
		Now:  time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC),
		Tx:   domain.Transaction{ID: "tx-1", Amount: domain.FromRubles(30000), Type: domain.TxKindIncoming},
		Kind: domain.IncomeSalary,
		Rule: domain.SavingsRule{
			GoalAmount: domain.FromRubles(36000), SavePercent: 10,
			MinIncome: domain.FromRubles(3000), MinTransfer: domain.DefaultMinTransfer,
		},
		Client: domain.Client{
			CardBalance: domain.FromRubles(32000), // 2 000 ₽ было + 30 000 ₽ пришло
			Mode:        domain.ModeSmart,
		},
		Profile: domain.Profile{
			Trust:    domain.TrustLevelHigh,
			Expected: map[domain.IncomeKind]domain.Money{domain.IncomeSalary: domain.FromRubles(30000)},
		},
		Upcoming: domain.FromRubles(20000), // платежи и траты до следующего дохода
	}
}

// Шаг 1. Когда откладывать нельзя совсем — сразу «пропустить» с понятной причиной.
func TestDecideSkips(t *testing.T) {
	tests := []struct {
		name       string
		edit       func(in *Input) // чем этот случай отличается от обычного
		wantReason string          // какое слово должно быть в причине для клиента
	}{
		{"счёт арестован", func(in *Input) { in.Client.Arrested = true }, "арестован"},
		{"копилка на паузе", func(in *Input) { in.Rule.Paused = true }, "пауз"},
		{"цель достигнута", func(in *Input) { in.Client.SavingsBalance = domain.FromRubles(36000) }, "цель"},
		{"не доход", func(in *Input) { in.Kind = domain.IncomeNone }, "не доход"},
		{"поступление меньше порога клиента", func(in *Input) { in.Tx.Amount = domain.FromRubles(2000) }, "меньше 3 000 ₽ не трогаем"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := anya()
			tt.edit(&in)
			got := Decide(in, cfg)

			if got.Action != domain.DecisionSkip || got.Amount != 0 || got.Status != domain.DecisionSkipped {
				t.Errorf("Decide() = %s %v, статус %s; want skip 0, статус skipped", got.Action, got.Amount, got.Status)
			}
			if !hasReason(got.Reason, tt.wantReason) {
				t.Errorf("причина %q должна содержать %q", got.Reason, tt.wantReason)
			}
		})
	}
}

// Шаг 2. Простой режим: клиент отказался от анализа операций. Откладываем фиксированный процент
// с каждого входящего поступления — без классификатора, профиля и вопросов, как обычное автопополнение.
func TestDecideSimpleMode(t *testing.T) {
	simple := func() Input {
		in := anya()
		in.Client.Mode = domain.ModeSimple
		in.Kind = "" // в простом режиме классификатор не запускается
		in.Tx.CounterpartyKind = domain.CounterpartyLegal
		return in
	}

	tests := []struct {
		name       string
		edit       func(in *Input)
		wantAction domain.DecisionAction
		wantAmount domain.Money
		wantReason string
	}{
		{"10% с поступления", func(in *Input) {}, domain.DecisionAuto, domain.FromRubles(3000), "10%"},
		{"предстоящие платежи не учитываем — анализа нет", func(in *Input) { in.Upcoming = domain.FromRubles(1_000_000) },
			domain.DecisionAuto, domain.FromRubles(3000), "10%"},
		{"перевод из копилки не трогаем — иначе цикл", func(in *Input) {
			in.Tx.Type, in.Tx.CounterpartyKind = domain.TxKindInternal, domain.CounterpartySelf
		}, domain.DecisionSkip, 0, "между вашими счетами"},
		{"перевод со своей карты другого банка не трогаем", func(in *Input) { in.Tx.CounterpartyKind = domain.CounterpartySelf },
			domain.DecisionSkip, 0, "между вашими счетами"},
		{"поступление меньше порога клиента", func(in *Input) { in.Tx.Amount = domain.FromRubles(2000) },
			domain.DecisionSkip, 0, "меньше 3 000 ₽ не трогаем"},
		{"10% меньше минимального перевода", func(in *Input) {
			in.Rule.MinIncome, in.Tx.Amount = 0, domain.FromRubles(900) // 10% = 90 ₽ < 100 ₽
		}, domain.DecisionSkip, 0, "меньше 100 ₽"},
		{"на карте меньше 10% — переводим сколько есть", func(in *Input) { in.Client.CardBalance = domain.FromRubles(1500) },
			domain.DecisionAuto, domain.FromRubles(1500), "10%"},
		{"арест важнее простого режима", func(in *Input) { in.Client.Arrested = true },
			domain.DecisionSkip, 0, "арестован"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := simple()
			tt.edit(&in)
			got := Decide(in, cfg)

			if got.Action != tt.wantAction || got.Amount != tt.wantAmount {
				t.Errorf("Decide() = %s %v, want %s %v (причина: %q)", got.Action, got.Amount, tt.wantAction, tt.wantAmount, got.Reason)
			}
			wantStatus := domain.DecisionSkipped
			if tt.wantAction == domain.DecisionAuto {
				wantStatus = domain.DecisionPending // перевод ещё предстоит исполнить сервису
			}
			if got.Status != wantStatus {
				t.Errorf("статус %q, want %q", got.Status, wantStatus)
			}
			if !hasReason(got.Reason, tt.wantReason) {
				t.Errorf("причина %q должна содержать %q", got.Reason, tt.wantReason)
			}
		})
	}
}

// Шаг 3. Безопасная сумма: откладываем не больше, чем останется после обязательных платежей
// и повседневных трат до следующего дохода. Кредитный лимит — не свои деньги, его не считаем.
func TestDecideSafeAmount(t *testing.T) {
	tests := []struct {
		name       string
		edit       func(in *Input)
		cfg        Config
		wantAction domain.DecisionAction
		wantAmount domain.Money
		wantReason string
	}{
		{
			name:       "стабильный доход, денег хватает — откладываем 10% сами",
			edit:       func(in *Input) {}, // на карте 32 000, впереди 20 000 → безопасно 12 000 ≥ 3 000
			cfg:        cfg,
			wantAction: domain.DecisionAuto, wantAmount: domain.FromRubles(3000), wantReason: "обычн",
		},
		{
			name:       "буфер уменьшает безопасную сумму, но 10% всё ещё помещаются",
			edit:       func(in *Input) {}, // 32 000 − 20 000 − 9 000 = 3 000
			cfg:        Config{DropPct: 15, RaisePct: 15, Buffer: domain.FromRubles(9000)},
			wantAction: domain.DecisionAuto, wantAmount: domain.FromRubles(3000), wantReason: "обычн",
		},
		{
			name:       "из-за буфера остаётся меньше 100 ₽ — не откладываем",
			edit:       func(in *Input) {}, // 32 000 − 20 000 − 11 950 = 50 ₽
			cfg:        Config{DropPct: 15, RaisePct: 15, Buffer: domain.FromRubles(11950)},
			wantAction: domain.DecisionSkip, wantAmount: 0, wantReason: "обязательные платежи",
		},
		{
			name:       "впереди платежей больше, чем денег на карте — не откладываем",
			edit:       func(in *Input) { in.Upcoming = domain.FromRubles(40000) }, // 32 000 − 40 000 < 0
			cfg:        cfg,
			wantAction: domain.DecisionSkip, wantAmount: 0, wantReason: "обязательные платежи",
		},
		{
			name: "кредитный лимит не считается своими деньгами",
			edit: func(in *Input) {
				in.Upcoming = domain.FromRubles(31950)            // своих свободных — 50 ₽
				in.Client.CreditLimit = domain.FromRubles(100000) // а кредитки — 100 000 ₽
			},
			cfg:        cfg,
			wantAction: domain.DecisionSkip, wantAmount: 0, wantReason: "обязательные платежи",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := anya()
			tt.edit(&in)
			got := Decide(in, tt.cfg)

			if got.Action != tt.wantAction || got.Amount != tt.wantAmount {
				t.Errorf("Decide() = %s %v, want %s %v (причина: %q)", got.Action, got.Amount, tt.wantAction, tt.wantAmount, got.Reason)
			}
			if !hasReason(got.Reason, tt.wantReason) {
				t.Errorf("причина %q должна содержать %q", got.Reason, tt.wantReason)
			}
		})
	}
}

// Шаг 4. Холодный старт: истории мало (меньше 3 месяцев) — своим выводам ещё не доверяем
// и всегда спрашиваем клиента, даже если всё выглядит нормально.
func TestDecideColdStart(t *testing.T) {
	tests := []struct {
		name       string
		edit       func(in *Input)
		wantAction domain.DecisionAction
		wantAmount domain.Money
		wantReason string
	}{
		{"мало истории — спрашиваем, а не переводим сами", func(in *Input) { in.Profile.Trust = domain.TrustLevelLow },
			domain.DecisionAsk, domain.FromRubles(3000), "мы ещё изучаем ваш доход"},
		{"мало истории и впереди платежи — спрашиваем про безопасную сумму", func(in *Input) {
			in.Profile.Trust = domain.TrustLevelLow
			in.Upcoming = domain.FromRubles(31000) // безопасно 1 000 ₽
		}, domain.DecisionAsk, domain.FromRubles(1000), "мы ещё изучаем ваш доход"},
		{"мало истории, но платежи съедают всё — не спрашиваем впустую", func(in *Input) {
			in.Profile.Trust = domain.TrustLevelLow
			in.Upcoming = domain.FromRubles(40000)
		}, domain.DecisionSkip, 0, "обязательные платежи"},
		{"средний уровень доверия — это уже не холодный старт", func(in *Input) { in.Profile.Trust = domain.TrustLevelMid },
			domain.DecisionAuto, domain.FromRubles(3000), "обычн"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := anya()
			tt.edit(&in)
			got := Decide(in, cfg)
			check(t, got, tt.wantAction, tt.wantAmount, tt.wantReason)
		})
	}
}

// Шаг 5. Доход изменился или впереди платёж — не списываем молча, а спрашиваем.
// Обычный доход Ани — 30 000 ₽, порог отклонения — 15%.
func TestDecideIncomeChanges(t *testing.T) {
	rassrochka := &domain.Due{
		Counterparty: "Рассрочка за ноутбук", Amount: domain.FromRubles(6000),
		At: time.Date(2026, 10, 30, 10, 0, 0, 0, time.UTC), // через 5 дней
	}

	tests := []struct {
		name        string
		edit        func(in *Input)
		wantAction  domain.DecisionAction
		wantAmount  domain.Money
		wantReason  string
		wantOptions []domain.Money // nil — не проверяем
	}{
		{
			name: "демо income_drop: пришло 24 000 ₽, через 5 дней рассрочка — спросить про 1 000 ₽",
			edit: func(in *Input) {
				in.Tx.Amount, in.Client.CardBalance = domain.FromRubles(24000), domain.FromRubles(26000)
				in.Upcoming, in.Nearest = domain.FromRubles(25000), rassrochka // безопасно 1 000 ₽
			},
			wantAction: domain.DecisionAsk, wantAmount: domain.FromRubles(1000), wantReason: "ниже обычного на 20%",
			wantOptions: []domain.Money{domain.FromRubles(1000)},
		},
		{
			name: "доход упал на 20%, денег хватает — всё равно спрашиваем",
			edit: func(in *Input) {
				in.Tx.Amount, in.Client.CardBalance = domain.FromRubles(24000), domain.FromRubles(26000)
				in.Upcoming = domain.FromRubles(20000) // безопасно 6 000 ≥ 2 400
			},
			wantAction: domain.DecisionAsk, wantAmount: domain.FromRubles(2400), wantReason: "ниже обычного на 20%",
		},
		{
			name: "доход ровно на 15% ниже — ещё в пределах обычного",
			edit: func(in *Input) {
				in.Tx.Amount, in.Client.CardBalance = domain.FromRubles(25500), domain.FromRubles(27500)
			},
			wantAction: domain.DecisionAuto, wantAmount: domain.FromRubles(2550), wantReason: "обычн",
		},
		{
			name: "аванс сравниваем с авансами, а не с зарплатой — это не падение дохода",
			edit: func(in *Input) {
				in.Kind, in.Tx.Amount, in.Client.CardBalance = domain.IncomeAdvance, domain.FromRubles(10000), domain.FromRubles(12000)
				in.Upcoming = domain.FromRubles(5000)
				in.Profile.Expected[domain.IncomeAdvance] = domain.FromRubles(10000)
			},
			wantAction: domain.DecisionAuto, wantAmount: domain.FromRubles(1000), wantReason: "обычн",
		},
		{
			name: "впереди платёж урезал сумму — спрашиваем и называем платёж",
			edit: func(in *Input) {
				in.Upcoming, in.Nearest = domain.FromRubles(30500), rassrochka // безопасно 1 500 < 3 000
			},
			wantAction: domain.DecisionAsk, wantAmount: domain.FromRubles(1500), wantReason: "через 5 дн",
			wantOptions: []domain.Money{domain.FromRubles(1500)},
		},
		{
			name:       "впереди траты урезали сумму, конкретного платежа нет",
			edit:       func(in *Input) { in.Upcoming = domain.FromRubles(30500) },
			wantAction: domain.DecisionAsk, wantAmount: domain.FromRubles(1500), wantReason: "обязательные платежи",
		},
		{
			name: "премия — спрашиваем, не отложить ли больше",
			edit: func(in *Input) {
				in.Kind, in.Tx.Amount, in.Client.CardBalance = domain.IncomeBonus, domain.FromRubles(60000), domain.FromRubles(62000)
			},
			wantAction: domain.DecisionAsk, wantAmount: domain.FromRubles(6000), wantReason: "премия",
			wantOptions: []domain.Money{domain.FromRubles(6000), domain.FromRubles(12000)}, // 10% или вдвое больше
		},
		{
			name: "доход вырос на 20% — откладываем и предлагаем поднять процент",
			edit: func(in *Input) {
				in.Tx.Amount, in.Client.CardBalance = domain.FromRubles(36000), domain.FromRubles(38000)
			},
			wantAction: domain.DecisionSuggestRaise, wantAmount: domain.FromRubles(3600), wantReason: "вырос на 20%",
		},
		{
			name: "доход ровно на 15% выше — ещё в пределах обычного",
			edit: func(in *Input) {
				in.Tx.Amount, in.Client.CardBalance = domain.FromRubles(34500), domain.FromRubles(36500)
			},
			wantAction: domain.DecisionAuto, wantAmount: domain.FromRubles(3450), wantReason: "обычн",
		},
		{
			name: "доход вырос, но доверие среднее — не предлагаем, а спрашиваем",
			edit: func(in *Input) {
				in.Profile.Trust = domain.TrustLevelMid
				in.Tx.Amount, in.Client.CardBalance = domain.FromRubles(36000), domain.FromRubles(38000)
			},
			wantAction: domain.DecisionAsk, wantAmount: domain.FromRubles(3600), wantReason: "выше обычного на 20%",
		},
		{
			name: "первый доход такого вида — сравнить не с чем, переводим",
			edit: func(in *Input) {
				in.Kind, in.Tx.Amount, in.Client.CardBalance = domain.IncomeFreelance, domain.FromRubles(5000), domain.FromRubles(25000)
				in.Upcoming = domain.FromRubles(10000)
			},
			wantAction: domain.DecisionAuto, wantAmount: domain.FromRubles(500), wantReason: "обычн",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := anya()
			tt.edit(&in)
			got := Decide(in, cfg)
			check(t, got, tt.wantAction, tt.wantAmount, tt.wantReason)
			if tt.wantOptions != nil && !slices.Equal(got.Options, tt.wantOptions) {
				t.Errorf("варианты %v, want %v", got.Options, tt.wantOptions)
			}
		})
	}
}

// Одинаковый вход — одинаковое решение: демо-сценарии должны повторяться.
func TestDecideIsDeterministic(t *testing.T) {
	in := anya()
	first := Decide(in, cfg)
	for range 100 {
		if got := Decide(anya(), cfg); got.Action != first.Action || got.Amount != first.Amount || got.Reason != first.Reason {
			t.Fatalf("решение изменилось: %+v vs %+v", got, first)
		}
	}
}

// check — общая проверка решения: действие, сумма, статус и причина.
func check(t *testing.T, got domain.Decision, action domain.DecisionAction, amount domain.Money, reason string) {
	t.Helper()
	if got.Action != action || got.Amount != amount {
		t.Errorf("Decide() = %s %v, want %s %v (причина: %q)", got.Action, got.Amount, action, amount, got.Reason)
	}
	wantStatus := domain.DecisionPending // auto, ask, suggest_raise — ждут перевода или ответа
	if action == domain.DecisionSkip {
		wantStatus = domain.DecisionSkipped
	}
	if got.Status != wantStatus {
		t.Errorf("статус %q, want %q", got.Status, wantStatus)
	}
	if !hasReason(got.Reason, reason) {
		t.Errorf("причина %q должна содержать %q", got.Reason, reason)
	}
}

// hasReason — есть ли фраза в причине для клиента: без учёта регистра, «е» и «ё» — одна буква.
func hasReason(reason, want string) bool {
	norm := func(s string) string { return strings.ReplaceAll(strings.ToLower(s), "ё", "е") }
	return strings.Contains(norm(reason), norm(want))
}

// Расшифровка: из чего сложилась сумма — чтобы клиент видел, на что у него впереди траты.
func TestDecideExplanation(t *testing.T) {
	at := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 7, 0, 0, 0, time.UTC) }
	// income_drop из демо: пришло 24 000 ₽, на карте 26 000 ₽, впереди 25 000 ₽.
	drop := func() Input {
		in := anya()
		in.Now = at(time.October, 20)
		in.Tx.Amount, in.Client.CardBalance = domain.FromRubles(24000), domain.FromRubles(26000)
		in.Profile.NextIncomeAt = at(time.November, 19)
		in.Profile.DailySpending = domain.FromRubles(600)
		in.Dues = []domain.Due{
			{Counterparty: "Рассрочка за ноутбук", Amount: domain.FromRubles(6000), At: at(time.October, 25)},
			{Counterparty: "Яндекс Плюс", Amount: domain.FromRubles(500), At: at(time.November, 3)},
			{Counterparty: "МТС", Amount: domain.FromRubles(500), At: at(time.November, 10)},
		}
		in.Spending, in.Days = domain.FromRubles(18000), 30
		in.Upcoming = domain.FromRubles(25000)
		in.Nearest = &in.Dues[0]
		return in
	}

	t.Run("демо income_drop: цифры и список трат", func(t *testing.T) {
		got := Decide(drop(), cfg).Explanation
		if got == nil {
			t.Fatal("у решения нет расшифровки")
		}
		if got.CardBalance != domain.FromRubles(26000) || got.UpcomingTotal != domain.FromRubles(25000) ||
			got.SafeAmount != domain.FromRubles(1000) || got.Buffer != 0 {
			t.Errorf("на карте %v, впереди %v, буфер %v, безопасно %v; want 26 000, 25 000, 0, 1 000",
				got.CardBalance, got.UpcomingTotal, got.Buffer, got.SafeAmount)
		}
		if !got.Until.Equal(at(time.November, 19)) {
			t.Errorf("считали до %s, want до 19.11 — следующий доход", got.Until.Format("02.01"))
		}

		want := []domain.UpcomingItem{
			{Title: "Рассрочка за ноутбук", Amount: domain.FromRubles(6000)},
			{Title: "Яндекс Плюс", Amount: domain.FromRubles(500)},
			{Title: "МТС", Amount: domain.FromRubles(500)},
			{Title: "Повседневные траты", Amount: domain.FromRubles(18000), PerDay: domain.FromRubles(600), Days: 30},
		}
		if len(got.Upcoming) != len(want) {
			t.Fatalf("трат в списке %d, want %d: %+v", len(got.Upcoming), len(want), got.Upcoming)
		}
		for i, w := range want {
			g := got.Upcoming[i]
			if g.Title != w.Title || g.Amount != w.Amount || g.PerDay != w.PerDay || g.Days != w.Days {
				t.Errorf("трата %d = %+v, want %+v", i, g, w)
			}
		}
		if g := got.Upcoming[0]; g.At == nil || !g.At.Equal(at(time.October, 25)) {
			t.Errorf("у платежа должна быть дата 25.10, получено %v", g.At)
		}
		if got.Upcoming[3].At != nil {
			t.Error("у повседневных трат нет одной даты")
		}

		for _, phrase := range []string{
			"можно безопасно отложить до 1 000 ₽",
			"до 19 ноября предстоят траты на 25 000 ₽",
			"рассрочка за ноутбук — 6 000 ₽, 25 октября",
			"мтс — 500 ₽, 10 ноября",
			"повседневные траты — 18 000 ₽ (≈ 600 ₽ в день × 30 дн.)",
			"на карте 26 000 ₽ − 25 000 ₽ = 1 000 ₽",
		} {
			if !hasReason(got.Summary, phrase) {
				t.Errorf("в тексте нет %q:\n%s", phrase, got.Summary)
			}
		}
	})

	t.Run("арифметика сходится: безопасно — это не сколько откладываем, а сколько можно", func(t *testing.T) {
		in := drop()
		in.Tx.Amount, in.Client.CardBalance = domain.FromRubles(30000), domain.FromRubles(32000) // как в stable
		got := Decide(in, cfg)
		if got.Action != domain.DecisionAuto || got.Amount != domain.FromRubles(3000) || got.Explanation == nil {
			t.Fatalf("Decide() = %s %v, want auto 3 000 ₽ с расшифровкой", got.Action, got.Amount)
		}
		e := got.Explanation
		if e.SafeAmount != e.CardBalance-e.UpcomingTotal-e.Buffer {
			t.Errorf("на карте %v − впереди %v − буфер %v ≠ безопасно %v", e.CardBalance, e.UpcomingTotal, e.Buffer, e.SafeAmount)
		}
		if !hasReason(e.Summary, "можно безопасно отложить до 7 000 ₽") || !hasReason(e.Summary, "= 7 000 ₽") {
			t.Errorf("в тексте должна быть безопасная сумма 7 000 ₽:\n%s", e.Summary)
		}
	})

	t.Run("пропуск из-за платежей — тоже объясняем", func(t *testing.T) {
		in := drop()
		in.Upcoming = domain.FromRubles(30000) // больше, чем на карте
		got := Decide(in, cfg)
		if got.Action != domain.DecisionSkip || got.Explanation == nil {
			t.Fatalf("Decide() = %s, расшифровка %v; want skip с расшифровкой", got.Action, got.Explanation)
		}
		if got.Explanation.SafeAmount != 0 || !hasReason(got.Explanation.Summary, "сейчас безопасно отложить нечего") {
			t.Errorf("безопасно %v, текст:\n%s", got.Explanation.SafeAmount, got.Explanation.Summary)
		}
	})

	t.Run("буфер виден в расчёте", func(t *testing.T) {
		got := Decide(drop(), Config{DropPct: 15, RaisePct: 15, Buffer: domain.FromRubles(500)}).Explanation
		if got == nil || got.SafeAmount != domain.FromRubles(500) || !hasReason(got.Summary, "− неприкосновенный остаток 500 ₽ = 500 ₽") {
			t.Errorf("расшифровка с буфером: %+v", got)
		}
	})

	t.Run("арест, пауза, не доход, простой режим — считать нечего, расшифровки нет", func(t *testing.T) {
		for _, edit := range []func(in *Input){
			func(in *Input) { in.Client.Arrested = true },
			func(in *Input) { in.Rule.Paused = true },
			func(in *Input) { in.Kind = domain.IncomeNone },
			func(in *Input) { in.Client.Mode = domain.ModeSimple },
		} {
			in := drop()
			edit(&in)
			if got := Decide(in, cfg); got.Explanation != nil {
				t.Errorf("%s: расшифровки быть не должно, а есть: %s", got.Reason, got.Explanation.Summary)
			}
		}
	})
}
