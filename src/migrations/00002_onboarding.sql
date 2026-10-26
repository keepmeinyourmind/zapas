-- +goose Up
-- Анкета при подключении: согласия, возраст, доходы и обязательные платежи.
CREATE TABLE onboarding (
    client_id        TEXT PRIMARY KEY REFERENCES clients(id) ON DELETE CASCADE,
    data_consent_at  TIMESTAMPTZ NOT NULL,
    analysis_consent BOOLEAN NOT NULL,
    birth_date       DATE    NOT NULL,
    monthly_income   BIGINT  NOT NULL CHECK (monthly_income >= 0),
    income_type      TEXT    NOT NULL,
    income_stability TEXT    NOT NULL,
    income_months    INT     NOT NULL CHECK (income_months >= 0),
    has_loans        BOOLEAN NOT NULL,
    monthly_payments BIGINT  NOT NULL CHECK (monthly_payments >= 0),
    score            INT     NOT NULL,
    eligible         BOOLEAN NOT NULL,
    reasons          JSONB   NOT NULL DEFAULT '[]',
    updated_at       TIMESTAMPTZ NOT NULL
);

-- Режим копилки и арест счёта.
ALTER TABLE clients
    ADD COLUMN mode              TEXT    NOT NULL DEFAULT 'smart',
    ADD COLUMN arrested          BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN arrest_changed_at TIMESTAMPTZ;

-- События от банка (арест и снятие ареста); id — защита от повторов.
CREATE TABLE account_events (
    id          TEXT PRIMARY KEY,
    client_id   TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL
);

-- Тихие часы клиента.
ALTER TABLE savings_rules
    ADD COLUMN quiet_from TEXT NOT NULL DEFAULT '23:00',
    ADD COLUMN quiet_to   TEXT NOT NULL DEFAULT '09:00',
    ADD COLUMN timezone   TEXT NOT NULL DEFAULT 'Europe/Moscow';

-- Когда отправить уведомление о решении (с учётом тихих часов).
ALTER TABLE decisions ADD COLUMN notify_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE decisions DROP COLUMN notify_at;
ALTER TABLE savings_rules DROP COLUMN timezone, DROP COLUMN quiet_to, DROP COLUMN quiet_from;
DROP TABLE account_events;
ALTER TABLE clients DROP COLUMN arrest_changed_at, DROP COLUMN arrested, DROP COLUMN mode;
DROP TABLE onboarding;
