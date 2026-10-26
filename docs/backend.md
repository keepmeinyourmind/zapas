# Бэкенд «Запас»: архитектура и план реализации

Документ для разработчика бэкенда. Отвечает на три вопроса: **из чего состоит проект**, **как части связаны между собой** и **в каком порядке их делать**. Логика продукта (что именно решает движок и почему) — в `docs/plan.md`, здесь — как это устроено в коде.

Сдача — 31 октября. Ниже календарь с 22 октября.

---

## 1. Архитектура одной картинкой

```
            HTTP (фронтенд, Swagger, пульт сценариев)
                              │
                      internal/api            ← разбирает JSON, вызывает сервис, отдаёт JSON
                              │
                    internal/service          ← сценарий обработки, транзакции БД, переводы
            ┌─────────────┬───┴────────┬──────────────┐
   internal/classifier  internal/profile  internal/engine   ← чистая логика: без БД, без HTTP, без time.Now()
            └─────────────┴───┬────────┴──────────────┘
                       internal/domain                 ← общие типы, все пакеты зависят от него
                              │
                     internal/storage                  ← SQL, PostgreSQL
```

**Главное правило зависимостей: стрелки идут только вниз.**

- `domain` ни от кого не зависит.
- `classifier`, `profile`, `engine` зависят только от `domain`. Они не знают про БД, HTTP и текущее время — получают всё параметрами. Поэтому их легко тестировать.
- `storage` знает про `domain` и PostgreSQL, но не знает про бизнес-логику.
- `service` — единственное место, которое склеивает логику и БД.
- `api` знает только про `service` и `domain`.
- `main.go` создаёт все объекты и связывает их («ручное внедрение зависимостей»).

Если в каком-то пакете хочется сделать импорт «вверх» (например, из `engine` в `storage`) — значит, код лежит не там.

---

## 2. Структура проекта

```
zapas/
├── docker-compose.yml
├── docs/
│   ├── plan.md                    # логика продукта, сценарии, пограничные случаи
│   └── backend.md                 # этот файл
└── src/                           # Go-модуль бэкенда
    ├── go.mod
    ├── Dockerfile
    ├── cmd/
    │   └── server/
    │       └── main.go            # конфиг → БД → миграции → сборка зависимостей → HTTP-сервер
    ├── api/
    │   └── openapi.yaml           # контракт с фронтендом
    ├── migrations/
    │   ├── embed.go               # встраивает .sql в бинарник
    │   └── 00001_init.sql         # схема БД
    └── internal/
        ├── config/
        │   └── config.go          # чтение переменных окружения
        ├── domain/
        │   ├── types.go           # Money, Transaction, Rule, Decision, Transfer (уже есть)
        │   ├── types_test.go
        │   ├── client.go          # Client, Profile, RecurringPayment (добавить)
        │   └── errors.go          # ErrNotFound, ErrDecisionClosed и т.д.
        ├── clock/
        │   └── clock.go           # управляемое время для демо
        ├── classifier/
        │   ├── classifier.go      # Classify(tx, history, confirmed) IncomeKind
        │   └── classifier_test.go
        ├── profile/
        │   ├── profile.go         # Build(...) Profile — ожидаемый доход, уровень доверия
        │   ├── recurring.go       # DetectRecurring, Upcoming — обязательные платежи
        │   └── profile_test.go
        ├── engine/
        │   ├── engine.go          # Input, Config, Decide(Input) Decision
        │   ├── reason.go          # тексты причин на русском
        │   └── engine_test.go
        ├── storage/
        │   ├── store.go           # пул соединений, InTx
        │   ├── clients.go         # клиенты и балансы
        │   ├── rules.go
        │   ├── transactions.go
        │   ├── decisions.go
        │   └── transfers.go
        ├── service/
        │   ├── service.go         # Service, конструктор
        │   ├── process.go         # ProcessTransaction — главный конвейер
        │   ├── answer.go          # AnswerDecision — ответ клиента на вопрос
        │   ├── expire.go          # ExpireDue — таймауты
        │   ├── transfer.go        # executeTransfer — перевод с карты в копилку
        │   ├── queries.go         # чтение для API: savings, decisions, transactions
        │   └── service_test.go    # интеграционные тесты с настоящей БД
        ├── scenarios/
        │   ├── anya.go            # данные Ани: история + ключевое событие для каждого сценария
        │   └── runner.go          # сброс клиента, загрузка истории, запуск события
        └── api/
            ├── server.go          # роутинг, middleware (CORS, логирование)
            ├── handlers.go        # обработчики эндпоинтов
            ├── dto.go             # структуры запросов и ответов
            └── errors.go          # единый формат ошибок
```

Пакет `internal/transfer` из старого плана не нужен: перевод — часть сервиса (`service/transfer.go`), потому что он должен выполняться в той же транзакции БД, что и решение.

---

## 3. Зависимости (`go.mod`)

Внешних библиотек всего три:

```bash
cd src
go get github.com/jackc/pgx/v5            # драйвер PostgreSQL и пул соединений
go get github.com/pressly/goose/v3        # миграции
go get github.com/google/uuid             # ID для решений и переводов
```

Всё остальное — стандартная библиотека: `net/http`, `encoding/json`, `log/slog`, `time`, `testing`.

**Внимание к версии Go.** Свежие версии goose требуют новый Go (последняя на момент написания — Go 1.26 и выше), и `go get` может сам поднять строку `go` в `go.mod`. Проверь `go version` и строку `go` в `go.mod` после `go get`: версия образа в `Dockerfile` (раздел 6) должна быть не ниже неё, иначе сборка в Docker упадёт.

---

## 4. Слои подробно

### 4.1. `domain` — общие типы

`types.go` уже есть. Добавить `client.go`:

```go
package domain

import "time"

// Client — клиент и его балансы.
type Client struct {
	ID             string
	Name           string
	CardBalance    Money // собственные деньги на карте
	CreditLimit    Money // доступный кредит; в расчёте безопасной суммы НЕ участвует
	SavingsBalance Money // баланс копилки
}

// Profile — что система знает о доходе клиента. Строит пакет profile, использует engine.
type Profile struct {
	Confidence    Confidence
	MonthsOfData  int
	Expected      map[IncomeKind]Money // ожидаемая сумма для каждого вида дохода (медиана)
	NextIncomeAt  time.Time            // когда ждём следующий доход
	DailySpending Money                // средние повседневные траты в день
}

// ExpectedFor возвращает ожидаемую сумму для вида дохода; 0 — если не знаем.
func (p Profile) ExpectedFor(k IncomeKind) Money { return p.Expected[k] }

// RecurringPayment — регулярный обязательный платёж (рассрочка, связь, подписка).
type RecurringPayment struct {
	Counterparty string
	Amount       Money
	DayOfMonth   int
}
```

`errors.go`:

```go
package domain

import "errors"

var (
	ErrNotFound       = errors.New("не найдено")
	ErrDecisionClosed = errors.New("решение уже принято")
	ErrDecisionExpired = errors.New("время на ответ истекло")
	ErrInvalidAnswer  = errors.New("такой вариант ответа недоступен")
)
```

Сервис возвращает эти ошибки, API превращает их в HTTP-коды (раздел 4.9).

### 4.2. `clock` — управляемое время

Движку время передаётся параметром, но сервису нужно «сейчас»: для таймаутов, для истории за 6 месяцев, для демо-кнопки «прошли сутки». Поэтому время — зависимость:

```go
package clock

type Clock interface{ Now() time.Time }

// Demo — часы, которые можно установить и промотать. В продакшене был бы time.Now().
type Demo struct {
	mu     sync.Mutex
	offset time.Duration
}

func (c *Demo) Now() time.Time {
	c.mu.Lock(); defer c.mu.Unlock()
	return time.Now().Add(c.offset)
}
func (c *Demo) Set(t time.Time)            { /* offset = t - time.Now() */ }
func (c *Demo) Advance(d time.Duration)    { /* offset += d */ }
```

Сценарий выставляет фиксированную дату (`Set`), поэтому результат всегда один и тот же. Кнопка «прошли сутки» вызывает `Advance(24 * time.Hour)`.

### 4.3. `classifier` — доход ли это

```go
// Classify определяет вид поступления.
//   tx        — новая операция
//   history   — прошлые операции клиента (уже классифицированные), отсортированы по времени
//   confirmed — отправители-физлица, которых клиент подтвердил как источник дохода
func Classify(tx domain.Transaction, history []domain.Transaction, confirmed map[string]bool) domain.IncomeKind
```

Порядок проверок (каждая — `if ... return`):

1. `tx.Type != TxIncoming` → `NotIncome`. Сюда попадают `internal` (в т.ч. из копилки — **защита от цикла**), `refund`, `loan`, `outgoing`.
2. `SenderSelf` или `SenderBank` → `NotIncome`.
3. `SenderPerson`: если `confirmed[tx.Counterparty]` → `IncomeFreelance`, иначе `NotIncome`.
4. `SenderLegal`: смотрим прошлые поступления от того же `Counterparty` за 40 дней.
   - нет ни одного → `IncomeSalary` (первая зарплата нового клиента);
   - сумма больше 1,5 × медианы прошлых выплат этого отправителя → `IncomeBonus`;
   - сумма меньше 0,7 × максимума прошлых выплат за месяц → `IncomeAdvance`;
   - иначе → `IncomeSalary`.

Стипендию в MVP не выделяем отдельно: для движка она ведёт себя как зарплата. Если нужно — по списку ИНН вузов.

Вспомогательная функция `median([]Money) Money` понадобится и в `profile` — положи её в `domain` (например, `domain.Median`).

### 4.4. `profile` — что мы знаем о клиенте

```go
// Build строит профиль по истории (последние 180 дней, уже классифицированные операции).
func Build(history []domain.Transaction, kinds map[string]domain.IncomeKind, rule domain.Rule, now time.Time) domain.Profile

// DetectRecurring находит регулярные списания: один получатель, ≥2 раз, сумма ±10%, интервал 25–35 дней.
func DetectRecurring(history []domain.Transaction) []domain.RecurringPayment

// Upcoming — сколько денег понадобится от now до until:
// регулярные платежи, попадающие в этот интервал + DailySpending × число дней.
func Upcoming(p domain.Profile, payments []domain.RecurringPayment, now, until time.Time) (total domain.Money, nearest *domain.RecurringPayment)
```

`kinds` — вид дохода для каждой транзакции (ID → вид), хранится в БД в колонке `income_kind`. Можно вместо отдельной map добавить поле `Kind` в `domain.Transaction` — так даже проще.

Как `Build` считает:
- `MonthsOfData` — сколько календарных месяцев, в которых был хотя бы один доход.
- `Confidence`: `<2` → Low, `2–3` → Medium, `>3` → High.
- `Expected[kind]` — медиана сумм этого вида за 6 месяцев. При `Low` для `IncomeSalary` берём `rule.DeclaredIncome`.
- `NextIncomeAt` — дата последней зарплаты + 30 дней (нет зарплаты → `now` + 30 дней).
- `DailySpending` — сумма исходящих операций за последние 30 дней **без** регулярных платежей, делённая на 30.

`nearest` из `Upcoming` нужен движку для текста причины: «через 5 дней платёж по рассрочке».

### 4.5. `engine` — решение

```go
type Config struct {
	DropThreshold  float64       // 0.15 — доход ниже ожидаемого на 15% → вопрос
	RaiseThreshold float64       // 0.15 — выше на 15% → предложить поднять процент
	Buffer         domain.Money  // неприкосновенный остаток на карте, например 0 или 1 000 ₽
	AskTTL         time.Duration // 24 часа на ответ
}

type Input struct {
	Now      time.Time
	Tx       domain.Transaction
	Kind     domain.IncomeKind
	Rule     domain.Rule
	Profile  domain.Profile
	Client   domain.Client             // CardBalance уже включает этот доход
	Upcoming domain.Money              // из profile.Upcoming
	Nearest  *domain.RecurringPayment  // ближайший обязательный платёж, может быть nil
}

// Decide — чистая функция: одинаковый Input → одинаковое решение.
// Заполняет Action, Amount, Options, Reason, Status, ExpiresAt. ID и CreatedAt ставит сервис.
func Decide(in Input, cfg Config) domain.Decision
```

Порядок проверок — в `docs/plan.md`, раздел 4. Опорные вычисления:

```go
safe := in.Client.CardBalance - in.Upcoming - cfg.Buffer   // CreditLimit не участвует
if safe < 0 { safe = 0 }
want   := in.Tx.Amount.Percent(in.Rule.Percent)
amount := min(want, safe)

expected := in.Profile.ExpectedFor(in.Kind)
dev := 0.0
if expected > 0 { dev = float64(in.Tx.Amount)/float64(expected) - 1 }
```

`Options` для вопроса: `[amount]`, и ещё `want`, если `want <= safe` и `want != amount`. Вариант «пропустить» в `Options` не кладём — он доступен всегда.
`Status`: `auto`/`suggest_raise` → `pending` до исполнения перевода сервисом, `ask` → `pending` + `ExpiresAt = Now + AskTTL`, `skip` → `skipped`.

`reason.go` — функции, собирающие тексты: `reasonDrop(dev float64)`, `reasonUpcoming(p *RecurringPayment, now time.Time)` и т.д. Тексты согласовать с PM: их показывает фронтенд.

**Этот пакет пишешь сам, начиная с тестов** (раздел 7).

### 4.6. `storage` — PostgreSQL

Идея: одни и те же методы работают и с пулом, и внутри транзакции.

```go
// DBTX — то, что умеют и *pgxpool.Pool, и pgx.Tx.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct{ pool *pgxpool.Pool }
type Repo struct{ db DBTX }   // все SQL-методы — на Repo

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Read — репозиторий без транзакции, для чтения из API.
func (s *Store) Read() *Repo { return &Repo{db: s.pool} }

// InTx выполняет fn в одной транзакции: ошибка → откат, иначе коммит.
func (s *Store) InTx(ctx context.Context, fn func(r *Repo) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil { return err }
	defer tx.Rollback(ctx) // после Commit ничего не делает
	if err := fn(&Repo{db: tx}); err != nil { return err }
	return tx.Commit(ctx)
}
```

Методы `Repo` (по файлам):

| Файл | Методы |
|---|---|
| `clients.go` | `GetClient`, `GetClientForUpdate` (`SELECT ... FOR UPDATE`), `UpdateBalances`, `UpsertClient`, `ResetClient` (для сценариев: удалить операции, решения, переводы, обнулить балансы) |
| `rules.go` | `GetRule` (нет → `domain.ErrNotFound`), `UpsertRule`, `SetPaused`, `SetPercent` |
| `transactions.go` | `InsertTransaction` → `(inserted bool, err)` через `ON CONFLICT (id) DO NOTHING`; `SetIncomeKind`; `MarkProcessed`; `ListTransactions(clientID, since)`; `ConfirmedSenders(clientID)` |
| `decisions.go` | `InsertDecision`, `GetDecisionForUpdate`, `GetDecisionBySourceTx`, `FindPending(clientID)`, `UpdateDecision` (статус, сумма, answered_at), `ListDecisions(clientID, limit)`, `ListExpiredPending(now)` |
| `transfers.go` | `InsertTransfer` → `(inserted bool, err)` через `ON CONFLICT (source_tx_id) DO NOTHING` |

Правило: `storage` не принимает решений. Только «положи / достань». Если в методе появился `if` про бизнес-логику — он должен уйти в `service` или `engine`.

`options` в БД — `JSONB`: при записи `json.Marshal(d.Options)`, при чтении `json.Unmarshal`.

### 4.7. `service` — конвейер

```go
type Service struct {
	store *storage.Store
	clock clock.Clock
	cfg   engine.Config
	log   *slog.Logger
}

func New(store *storage.Store, clk clock.Clock, cfg engine.Config, log *slog.Logger) *Service
```

**`ProcessTransaction(ctx, tx) (*domain.Decision, domain.IncomeKind, error)`** — всё в одном `InTx`:

```
1. inserted := repo.InsertTransaction(tx)
   если !inserted → вернуть решение по этой транзакции (GetDecisionBySourceTx) — повтор, ничего не делаем
2. client := repo.GetClientForUpdate(tx.ClientID)
   блокировка строки клиента: две транзакции одного клиента не обработаются параллельно
3. изменить баланс карты: incoming/refund/loan/internal → +amount, outgoing → −amount
4. history := repo.ListTransactions(clientID, now − 180 дней)
5. kind := classifier.Classify(tx, history, repo.ConfirmedSenders(clientID));  repo.SetIncomeKind
6. если !kind.IsIncome() → UpdateBalances, MarkProcessed, вернуть (nil, kind)
7. rule := repo.GetRule(clientID); нет правила → то же, что в п.6
8. prof := profile.Build(...); payments := profile.DetectRecurring(history)
   upcoming, nearest := profile.Upcoming(prof, payments, now, prof.NextIncomeAt)
9. dec := engine.Decide(engine.Input{...}, s.cfg); dec.ID = uuid; dec.CreatedAt = now
10. old := repo.FindPending(clientID); если есть → old.Status = expired, UpdateDecision(old)
11. repo.InsertDecision(dec)
12. если dec.Action ∈ {auto, suggest_raise} → s.executeTransfer(repo, &client, &dec, dec.Amount)
13. repo.UpdateBalances(client); repo.MarkProcessed(tx.ID)
```

**`executeTransfer(repo, client, dec, amount)`** (`transfer.go`) — внутри уже открытой транзакции:

```
1. amount = min(amount, client.CardBalance)       // повторная проверка: денег могло стать меньше
2. amount < rule.MinTransfer → dec.Status = skipped, причина дополняется, выход
3. inserted := repo.InsertTransfer(...)            // уникальный source_tx_id
   !inserted → перевод уже был, выход без изменения балансов
4. client.CardBalance -= amount; client.SavingsBalance += amount
5. dec.Amount = amount; dec.Status = executed; repo.UpdateDecision(dec)
```

**`AnswerDecision(ctx, decisionID, choice string, amount Money)`** (`answer.go`), в `InTx`:

```
1. dec := repo.GetDecisionForUpdate(id)            // нет → ErrNotFound
2. dec.Status != pending → ErrDecisionClosed
3. now > dec.ExpiresAt → dec.Status = expired, сохранить, ErrDecisionExpired
4. choice == "skip" → dec.Status = skipped, сохранить
5. amount нет в dec.Options → ErrInvalidAnswer
6. client := GetClientForUpdate; executeTransfer(repo, &client, &dec, amount); UpdateBalances
```

**`ExpireDue(ctx)`** (`expire.go`): `ListExpiredPending(now)` → каждому `Status = expired` с причиной «Нет ответа — ничего не откладывали». Выбираем самый безопасный вариант: пропуск.

**Чтение для API** (`queries.go`): `GetSavings(clientID)` (баланс, цель, прогресс в процентах, последний перевод), `ListDecisions`, `ListTransactions` с видом дохода. Без транзакций, через `store.Read()`.

### 4.8. `scenarios` — демо-данные Ани

```go
type Scenario struct {
	Name     string
	Now      time.Time            // фиксированное «сейчас» на момент ключевого события
	Client   domain.Client        // стартовые балансы
	Rule     domain.Rule
	History  []domain.Transaction // 3–6 месяцев операций
	Event    domain.Transaction   // ключевое событие
}

func All() map[string]Scenario   // stable, income_drop, self_transfer, new_client, income_rise
```

`runner.go`:

```
Run(ctx, name):
1. sc := All()[name]; нет → ErrNotFound
2. store.InTx: ResetClient(sc.Client.ID); UpsertClient(sc.Client); UpsertRule(sc.Rule)
3. для каждой операции истории: clock.Set(op.OccurredAt); service.ImportHistory(op)
   ImportHistory — как ProcessTransaction, но шаги 1–6 без движка: только сохранить, классифицировать, обновить баланс
4. client.CardBalance выставить стартовым значением сценария (история могла его изменить)
5. clock.Set(sc.Now); service.ProcessTransaction(sc.Event)
6. вернуть решение
```

Опорные числа, чтобы совпало с раскадровкой видео (подгони историю под них):

| | Значение |
|---|---|
| Остаток на карте до события | 2 000 ₽ |
| Регулярные платежи | рассрочка 6 000 ₽ (25-е), связь и подписки 1 000 ₽ |
| Повседневные траты | ≈ 600 ₽/день → около 18 000 ₽ за 30 дней |
| `Upcoming` до следующего дохода | ≈ 25 000 ₽ |
| `stable`: доход 30 000 ₽ | карта 32 000 → безопасно 7 000 → хотим 3 000 → `auto`, 3 000 ₽ |
| `income_drop`: доход 24 000 ₽ | карта 26 000 → безопасно 1 000 → хотим 2 400, отклонение −20% → `ask`, 1 000 ₽ |

Даты истории — фиксированные (например, с 1 апреля по 20 октября 2026), никакого `time.Now()` в сценариях.

### 4.9. `api` — HTTP

```go
type Server struct {
	svc    *service.Service
	runner *scenarios.Runner
	clock  *clock.Demo
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("PUT /api/clients/{id}/rule", s.putRule)
	mux.HandleFunc("PATCH /api/clients/{id}/rule", s.patchRule)
	mux.HandleFunc("POST /api/transactions", s.postTransaction)
	mux.HandleFunc("GET /api/clients/{id}/transactions", s.listTransactions)
	mux.HandleFunc("GET /api/clients/{id}/savings", s.getSavings)
	mux.HandleFunc("GET /api/clients/{id}/decisions", s.listDecisions)
	mux.HandleFunc("POST /api/decisions/{id}/answer", s.answerDecision)
	mux.HandleFunc("POST /api/scenarios/{name}/run", s.runScenario)
	mux.HandleFunc("POST /api/debug/advance-time", s.advanceTime)
	return withCORS(withLogging(mux))
}
```

`r.PathValue("id")` достаёт параметр из пути (Go 1.22+).

**Каждый обработчик делает одно и то же:**
1. Разобрать вход (`json.NewDecoder(r.Body).Decode(&req)`, `r.PathValue`).
2. Проверить простое (пустые поля, `rule.Validate()`).
3. Вызвать один метод сервиса.
4. Перевести результат в DTO и отдать `writeJSON(w, status, dto)`.

Бизнес-логики в обработчиках нет.

**DTO отдельно от domain** (`dto.go`): API-формат можно менять, не трогая логику. Например, в ответе решения удобно отдать и копейки, и готовую строку для экрана:

```json
{
  "id": "8f2c…",
  "action": "ask",
  "amount": 100000,
  "amount_text": "1 000 ₽",
  "options": [100000],
  "reason": "Доход ниже обычного на 20%, через 5 дней платёж по рассрочке 6 000 ₽",
  "status": "pending",
  "created_at": "2026-10-20T10:00:00+03:00",
  "expires_at": "2026-10-21T10:00:00+03:00"
}
```

**Ошибки** (`errors.go`) — всегда в одном формате:

```json
{ "error": { "code": "decision_closed", "message": "решение уже принято" } }
```

| Ошибка сервиса | HTTP |
|---|---|
| невалидный JSON, `rule.Validate()` | 400 `bad_request` |
| `domain.ErrNotFound` | 404 `not_found` |
| `ErrDecisionClosed`, `ErrDecisionExpired` | 409 `decision_closed` / `decision_expired` |
| `ErrInvalidAnswer` | 422 `invalid_answer` |
| всё остальное | 500 `internal` + запись в лог |

Сравнивать через `errors.Is(err, domain.ErrNotFound)`.

**CORS** — middleware, который ставит `Access-Control-Allow-Origin` (адрес фронтенда из конфига), `Access-Control-Allow-Methods`, `Access-Control-Allow-Headers: Content-Type` и отвечает `204` на `OPTIONS`.

### 4.10. `config` и `main.go` — как всё связывается

`config.go` читает переменные окружения со значениями по умолчанию:

| Переменная | По умолчанию |
|---|---|
| `DATABASE_URL` | `postgres://zapas:zapas@localhost:5433/zapas?sslmode=disable` |
| `HTTP_ADDR` | `:8080` |
| `CORS_ORIGIN` | `http://localhost:5173` |
| `DROP_THRESHOLD`, `RAISE_THRESHOLD` | `0.15` |
| `BUFFER_RUB` | `0` |

`main.go` — вся «проводка» в одном месте:

```go
func main() {
	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	cfg := config.Load()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil { log.Error("db", "err", err); os.Exit(1) }
	defer pool.Close()

	// миграции при старте: docker compose up — и база готова
	db := stdlib.OpenDBFromPool(pool)
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil { /* выход */ }
	if err := goose.Up(db, "."); err != nil { /* выход */ }

	clk := &clock.Demo{}
	store := storage.New(pool)
	svc := service.New(store, clk, cfg.Engine, log)
	runner := scenarios.NewRunner(store, svc, clk)
	srv := api.NewServer(svc, runner, clk, cfg.CORSOrigin, log)

	go func() { // таймауты вопросов
		t := time.NewTicker(time.Minute)
		for range t.C { svc.ExpireDue(ctx) }
	}()

	log.Info("listening", "addr", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, srv.Routes()); err != nil { /* выход */ }
}
```

`migrations/embed.go`:

```go
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
```

---

## 5. Схема БД — `migrations/00001_init.sql`

```sql
-- +goose Up
CREATE TABLE clients (
    id              TEXT PRIMARY KEY,
    name            TEXT   NOT NULL,
    card_balance    BIGINT NOT NULL DEFAULT 0,
    credit_limit    BIGINT NOT NULL DEFAULT 0,
    savings_balance BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE savings_rules (
    client_id          TEXT PRIMARY KEY REFERENCES clients(id) ON DELETE CASCADE,
    goal_name          TEXT    NOT NULL,
    goal_amount        BIGINT  NOT NULL CHECK (goal_amount > 0),
    term_months        INT     NOT NULL CHECK (term_months IN (6, 12, 24)),
    percent            INT     NOT NULL CHECK (percent BETWEEN 1 AND 50),
    min_income_trigger BIGINT  NOT NULL DEFAULT 0,
    min_transfer       BIGINT  NOT NULL DEFAULT 10000,
    declared_income    BIGINT  NOT NULL DEFAULT 0,
    paused             BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE transactions (
    id           TEXT PRIMARY KEY,                 -- ID от процессинга: защита от дублей
    client_id    TEXT   NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    amount       BIGINT NOT NULL CHECK (amount > 0),
    type         TEXT   NOT NULL,
    counterparty TEXT   NOT NULL DEFAULT '',
    sender_kind  TEXT   NOT NULL DEFAULT '',
    occurred_at  TIMESTAMPTZ NOT NULL,
    income_kind  TEXT,
    processed    BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX transactions_client_time ON transactions (client_id, occurred_at);

CREATE TABLE confirmed_senders (
    client_id    TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    counterparty TEXT NOT NULL,
    PRIMARY KEY (client_id, counterparty)
);

CREATE TABLE decisions (
    id           UUID PRIMARY KEY,
    client_id    TEXT   NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    source_tx_id TEXT   NOT NULL UNIQUE REFERENCES transactions(id) ON DELETE CASCADE,
    action       TEXT   NOT NULL,
    amount       BIGINT NOT NULL DEFAULT 0,
    options      JSONB  NOT NULL DEFAULT '[]',
    reason       TEXT   NOT NULL,
    status       TEXT   NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL,
    expires_at   TIMESTAMPTZ,
    answered_at  TIMESTAMPTZ
);
CREATE INDEX decisions_client_time ON decisions (client_id, created_at DESC);
CREATE INDEX decisions_pending ON decisions (expires_at) WHERE status = 'pending';

CREATE TABLE transfers (
    id           UUID PRIMARY KEY,
    decision_id  UUID   NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
    source_tx_id TEXT   NOT NULL UNIQUE,           -- одно поступление → максимум один перевод
    client_id    TEXT   NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    amount       BIGINT NOT NULL CHECK (amount > 0),
    created_at   TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE transfers;
DROP TABLE decisions;
DROP TABLE confirmed_senders;
DROP TABLE transactions;
DROP TABLE savings_rules;
DROP TABLE clients;
```

Регулярные платежи в отдельной таблице не храним: `profile.DetectRecurring` считает их по истории при каждой обработке. Для MVP этого хватает.

`ON DELETE CASCADE` нужен для сброса сценария: удалил клиента — удалилось всё связанное.

---

## 6. Docker

`src/Dockerfile`:

```dockerfile
# версия должна быть не ниже строки `go` в go.mod
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /server ./cmd/server

FROM alpine:3.20
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
```

`docker-compose.yml` в корне:

```yaml
services:
  db:
    image: postgres:16
    environment:
      POSTGRES_USER: zapas
      POSTGRES_PASSWORD: zapas
      POSTGRES_DB: zapas
    ports: ["5433:5432"]
    volumes: [pgdata:/var/lib/postgresql/data]
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U zapas"]
      interval: 2s
      retries: 15

  backend:
    build: ./src
    environment:
      DATABASE_URL: postgres://zapas:zapas@db:5432/zapas?sslmode=disable
      CORS_ORIGIN: http://localhost:5173
    ports: ["8080:8080"]
    depends_on:
      db: { condition: service_healthy }

  swagger:
    image: swaggerapi/swagger-ui
    environment:
      SWAGGER_JSON: /spec/openapi.yaml
    volumes: ["./src/api:/spec"]
    ports: ["8081:8080"]

volumes:
  pgdata:
```

Swagger UI будет на `localhost:8081`, API — на `localhost:8080`. Внутри Docker бэкенд обращается к базе по имени сервиса `db`, а не `localhost`.

---

## 7. Тесты

| Пакет | Что | Как |
|---|---|---|
| `engine` | каждая ветка `Decide` | table-driven, без БД, `Now` — фиксированная дата |
| `classifier` | перевод из копилки, возврат, кредит, свой перевод, неизвестное физлицо, аванс, премия, первая зарплата | table-driven |
| `profile` | уровень доверия по числу месяцев, `DetectRecurring` на истории Ани, `Upcoming` | table-driven |
| `service` | дубль транзакции → один перевод; новый доход закрывает старый вопрос; поздний ответ отклоняется; повторный ответ → `ErrDecisionClosed` | с настоящим Postgres из docker-compose |
| `scenarios` | каждый сценарий даёт ожидаемое действие и сумму из таблицы 4.8 | с Postgres; это и есть проверка демо |

Интеграционные тесты пропускаются, если нет базы:

```go
func testStore(t *testing.T) *storage.Store {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" { t.Skip("TEST_DATABASE_URL не задан") }
	// pgxpool.New, goose.Up, очистка таблиц
}
```

Запуск: `TEST_DATABASE_URL=postgres://zapas:zapas@localhost:5433/zapas?sslmode=disable go test ./...`

Шаблон теста движка — начни с главного сценария демо:

```go
func TestDecide(t *testing.T) {
	now := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	rule := domain.Rule{Percent: 10, GoalAmount: domain.Rub(36000), MinTransfer: domain.Rub(100)}
	prof := domain.Profile{Confidence: domain.ConfidenceHigh,
		Expected: map[domain.IncomeKind]domain.Money{domain.IncomeSalary: domain.Rub(30000)}}

	tests := []struct {
		name       string
		income     domain.Money
		card       domain.Money // баланс уже с доходом
		upcoming   domain.Money
		wantAction domain.Action
		wantAmount domain.Money
	}{
		{"стабильный доход", domain.Rub(30000), domain.Rub(32000), domain.Rub(25000), domain.ActionAuto, domain.Rub(3000)},
		{"доход упал на 20%", domain.Rub(24000), domain.Rub(26000), domain.Rub(25000), domain.ActionAsk, domain.Rub(1000)},
		// добавляй по кейсу на каждую ветку
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := engine.Input{Now: now, Kind: domain.IncomeSalary, Rule: rule, Profile: prof,
				Tx:       domain.Transaction{Amount: tt.income},
				Client:   domain.Client{CardBalance: tt.card},
				Upcoming: tt.upcoming}
			got := engine.Decide(in, engine.Config{DropThreshold: 0.15, RaiseThreshold: 0.15, AskTTL: 24 * time.Hour})
			if got.Action != tt.wantAction || got.Amount != tt.wantAmount {
				t.Errorf("got %s %v, want %s %v (reason: %s)", got.Action, got.Amount, tt.wantAction, tt.wantAmount, got.Reason)
			}
		})
	}
}
```

---

## 8. Порядок работы по дням

Каждый шаг заканчивается проверкой «как убедиться, что работает». Не переходи дальше, пока проверка не прошла.

| Дата | Что делаем | Проверка |
|---|---|---|
| **22 окт** | `config`, `migrations` (SQL выше), `storage/store.go`, `main.go` с миграциями и `/health`; `docker-compose.yml` с `db` | `go run ./cmd/server` → в логе миграции прошли; `psql` видит таблицы; `/health` отвечает |
| **23 окт** | `storage`: клиенты, правила, транзакции; `api`: `PUT rule`, `POST transactions` (пока только сохраняет), `GET transactions`; CORS; `openapi.yaml` → фронтенду | через Swagger или `curl`: правило сохраняется, повторный POST той же транзакции не создаёт дубль |
| **24 окт** | `classifier` + тесты; `domain.Median` | `go test ./internal/classifier/` зелёный, все кейсы из раздела 7 |
| **25 окт** | `profile`: `Build`, `DetectRecurring`, `Upcoming` + тесты; `clock` | на истории Ани находятся рассрочка и подписки, `Upcoming` ≈ 25 000 ₽ |
| **26–27 окт** | `engine`: `Decide`, `reason.go`, тесты на все ветки — **главный блок, пишешь сам** | `go test ./internal/engine/` зелёный, кейсы из таблицы 4.8 дают нужные суммы |
| **28 окт** | `service`: `ProcessTransaction`, `executeTransfer`, `AnswerDecision`, `ExpireDue`; `storage`: решения и переводы; API: решения, ответ, savings | `POST transactions` с зарплатой → в `GET decisions` есть решение, баланс копилки вырос |
| **29 окт** | `scenarios` + `runner`, `POST scenarios/{name}/run`, `advance-time`; `Dockerfile`, backend и swagger в compose; интеграционные тесты сценариев | `docker compose up` с нуля → все 5 сценариев дают результат из таблицы 4.8; фронтенд работает с живым API |
| **30 окт** | README; запуск с нуля на чужом компьютере; запись видео | эксперт по README поднимает проект за 5 минут |
| **31 окт** | багфиксы, ссылка на репозиторий PM | репозиторий публичный, открывается в инкогнито |

**Если отстаёшь**, режь в таком порядке:
1. `income_rise` и `suggest_raise`.
2. Премии в классификаторе.
3. Фоновый цикл таймаутов — оставить только `advance-time`, который вызывает `ExpireDue`.
4. `DetectRecurring` — платежи задать прямо в сценарии (`Scenario.Payments`).

Не режь никогда: движок с тестами, исключение переводов из копилки, идемпотентность, сценарии `income_drop` и `self_transfer`.

---

## 9. Что писать самому, а что можно поручить помощнику

| Сам (это суть проекта и то, что спросят на защите) | Можно поручить помощнику и разобрать результат |
|---|---|
| `engine` и его тесты | `config`, `migrations`, `Dockerfile`, `docker-compose.yml` |
| `classifier` и его тесты | `storage` (SQL-методы по таблице 4.6) |
| порядок шагов в `service/process.go` | обработчики `api`, DTO, CORS, формат ошибок |
| | `scenarios/anya.go` — генерация истории под опорные числа |
| | `openapi.yaml` по таблице эндпоинтов |

Код, который пишет помощник, обязательно читай целиком: на защите нужно уметь объяснить каждую строку.
