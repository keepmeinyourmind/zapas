package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/keepmeinyourmind/zapas/src/migrations"
)

// DBTX — то, что умеют и *pgxpool.Pool, и pgx.Tx.
// Благодаря этому одни и те же SQL-методы работают и с пулом, и внутри транзакции.
type DBTX interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct{ pool *pgxpool.Pool }

// Repo — набор SQL-методов поверх DBTX (пул или транзакция).
type Repo struct{ db DBTX }

func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// Read — репозиторий без транзакции, для чтения из API.
func (s *Store) Read() *Repo { return &Repo{db: s.pool} }

// InTx выполняет fn в одной транзакции: ошибка → откат, иначе коммит.
//
// Порядок блокировок везде один: сначала строка клиента (GetClientForUpdate или UPDATE clients),
// потом строки его решений. Иначе две транзакции ждут друг друга — взаимная блокировка (deadlock).
func (s *Store) InTx(ctx context.Context, fn func(r *Repo) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) // после Commit ничего не делает
	if err := fn(&Repo{db: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Migrate применяет все новые миграции из встроенной папки migrations.
func Migrate(pool *pgxpool.Pool) error {
	db := stdlib.OpenDBFromPool(pool)
	defer func(db *sql.DB) { _ = db.Close() }(db)

	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("диалект goose: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("миграции: %w", err)
	}
	return nil
}
