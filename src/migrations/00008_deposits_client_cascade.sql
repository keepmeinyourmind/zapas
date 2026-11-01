-- +goose Up
-- Как и прочая история клиента, ручные пополнения удаляются при сбросе демо.
ALTER TABLE deposits DROP CONSTRAINT deposits_client_id_fkey;
ALTER TABLE deposits ADD CONSTRAINT deposits_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES clients(id) ON DELETE CASCADE;

-- +goose Down
ALTER TABLE deposits DROP CONSTRAINT deposits_client_id_fkey;
ALTER TABLE deposits ADD CONSTRAINT deposits_client_id_fkey
    FOREIGN KEY (client_id) REFERENCES clients(id);
