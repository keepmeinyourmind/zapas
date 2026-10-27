-- +goose Up
-- Снятие денег из копилки на карту; id — защита от повторного запроса.
CREATE TABLE withdrawals (
    id         TEXT PRIMARY KEY,
    client_id  TEXT   NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    amount     BIGINT NOT NULL CHECK (amount > 0),
    created_at TIMESTAMPTZ NOT NULL
);

-- Платежи, которые клиент сам пометил как необязательные: под них не резервируем деньги.
CREATE TABLE recurring_exclusions (
    client_id    TEXT NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    counterparty TEXT NOT NULL,
    PRIMARY KEY (client_id, counterparty)
);

-- +goose Down
DROP TABLE recurring_exclusions;
DROP TABLE withdrawals;
