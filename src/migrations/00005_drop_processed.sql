-- +goose Up
-- Колонка не использовалась: обработку операции видно по income_kind и решению.
ALTER TABLE transactions DROP COLUMN processed;

-- +goose Down
ALTER TABLE transactions ADD COLUMN processed BOOLEAN NOT NULL DEFAULT FALSE;
