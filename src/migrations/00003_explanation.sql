-- +goose Up
-- Расшифровка решения: из чего сложилась безопасная сумма.
ALTER TABLE decisions ADD COLUMN explanation JSONB;

-- +goose Down
ALTER TABLE decisions DROP COLUMN explanation;
