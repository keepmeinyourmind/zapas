-- +goose Up
CREATE TABLE deposits (
    id TEXT PRIMARY KEY,
    client_id TEXT NOT NULL REFERENCES clients(id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deposits_client_id ON deposits(client_id);

COMMENT ON TABLE deposits IS 'История пополнений копилки с идемпотентностью по id';
COMMENT ON COLUMN deposits.id IS 'Уникальный идентификатор операции (для идемпотентности)';
COMMENT ON COLUMN deposits.client_id IS 'ID клиента';
COMMENT ON COLUMN deposits.amount IS 'Сумма пополнения в копейках (всегда положительная)';
COMMENT ON COLUMN deposits.created_at IS 'Время создания операции';

-- +goose Down
DROP TABLE IF EXISTS deposits;
