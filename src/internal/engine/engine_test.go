package engine

import (
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
			if !strings.Contains(strings.ToLower(got.Reason), tt.wantReason) {
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
			if !strings.Contains(strings.ToLower(got.Reason), tt.wantReason) {
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
			if !strings.Contains(strings.ToLower(got.Reason), tt.wantReason) {
				t.Errorf("причина %q должна содержать %q", got.Reason, tt.wantReason)
			}
		})
	}
}
