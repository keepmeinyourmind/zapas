-- +goose Up
CREATE TABLE clients (
    id              TEXT PRIMARY KEY,
    name            TEXT   NOT NULL,
    card_balance    BIGINT NOT NULL DEFAULT 0,
    credit_limit    BIGINT NOT NULL DEFAULT 0,
    savings_balance BIGINT NOT NULL DEFAULT 0
);

CREATE TABLE savings_rules (
    client_id       TEXT PRIMARY KEY REFERENCES clients(id) ON DELETE CASCADE,
    goal_name       TEXT    NOT NULL,
    goal_amount     BIGINT  NOT NULL CHECK (goal_amount > 0),
    term_months     INT     NOT NULL CHECK (term_months IN (6, 12, 24)),
    save_percent    INT     NOT NULL CHECK (save_percent BETWEEN 1 AND 50),
    min_income      BIGINT  NOT NULL DEFAULT 0,
    min_transfer    BIGINT  NOT NULL DEFAULT 10000,
    declared_income BIGINT  NOT NULL DEFAULT 0,
    paused          BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE transactions (
    id                TEXT PRIMARY KEY,                 -- ID от процессинга: защита от дублей
    client_id         TEXT   NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    amount            BIGINT NOT NULL CHECK (amount > 0),
    type              TEXT   NOT NULL,
    counterparty      TEXT   NOT NULL DEFAULT '',
    counterparty_kind TEXT   NOT NULL DEFAULT '',
    occurred_at       TIMESTAMPTZ NOT NULL,
    income_kind       TEXT,
    processed         BOOLEAN NOT NULL DEFAULT FALSE
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
