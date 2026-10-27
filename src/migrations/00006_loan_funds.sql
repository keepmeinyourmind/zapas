-- +goose Up
-- Кредитные деньги клиента на карте: тратятся после своих и в безопасной сумме не участвуют.
ALTER TABLE clients ADD COLUMN loan_funds BIGINT NOT NULL DEFAULT 0 CHECK (loan_funds >= 0);

-- +goose Down
ALTER TABLE clients DROP COLUMN loan_funds;
