package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/keepmeinyourmind/zapas/src/internal/clock"
	"github.com/keepmeinyourmind/zapas/src/internal/config"
	"github.com/keepmeinyourmind/zapas/src/internal/scenarios"
	"github.com/keepmeinyourmind/zapas/src/internal/service"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

// Проверяем полный HTTP-путь: пополнение → снятие → история, включая повторы запросов.
func TestOperationsHistory(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL не задан")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := storage.Migrate(pool); err != nil {
		t.Fatal(err)
	}
	store, clk := storage.New(pool), clock.New()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.New(store, config.Load(), clk, log)
	runner := scenarios.NewRunner(store, svc, clk)
	handler := NewServer(svc, runner, clk, "http://localhost:5173", log).Routes()
	request := func(t *testing.T, method, path, body string, want int) *httptest.ResponseRecorder {
		t.Helper()
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		if rec.Code != want {
			t.Fatalf("%s %s: HTTP %d, ожидали %d: %s", method, path, rec.Code, want, rec.Body)
		}
		return rec
	}
	for _, scenario := range []string{"stable", "income_drop"} {
		t.Run(scenario, func(t *testing.T) {
			rec := request(t, "POST", "/api/scenarios/"+scenario+"/run", "", 200)
			var result scenarios.Result
			if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Decision == nil {
				t.Fatal("нет решения")
			}
			amount := result.Decision.Amount
			if scenario == "income_drop" {
				request(t, "POST", "/api/decisions/"+result.Decision.ID+"/answer", `{"choice":"amount","amount":100000}`, 200)
			}
			clk.Advance(time.Hour)
			request(t, "POST", "/api/clients/anya/savings/withdraw", `{"id":"history-withdrawal","amount":10000}`, 201)
			request(t, "POST", "/api/clients/anya/savings/withdraw", `{"id":"history-withdrawal","amount":10000}`, 200)
			// Повтор банковского события также не добавляет ещё одно пополнение.
			sc, _ := scenarios.Get(scenario)
			tx := sc.Event
			tx.ClientID = "anya"
			body, err := json.Marshal(tx)
			if err != nil {
				t.Fatal(err)
			}
			request(t, "POST", "/api/transactions", string(body), 200)
			rec = request(t, "GET", "/api/clients/anya/operations", "", 200)
			var ops []storage.Operation
			if err := json.Unmarshal(rec.Body.Bytes(), &ops); err != nil {
				t.Fatal(err)
			}
			if len(ops) != 2 {
				t.Fatalf("история должна содержать 2 операции: %s", rec.Body)
			}
			if ops[0].Type != "withdrawal" || ops[0].ID != "history-withdrawal" || ops[0].Amount != 10000 || ops[0].DecisionID != nil {
				t.Fatalf("снятие: %+v", ops[0])
			}
			if ops[1].Type != "transfer_in" || ops[1].Amount != amount || ops[1].DecisionID == nil || *ops[1].DecisionID != result.Decision.ID {
				t.Fatalf("пополнение: %+v", ops[1])
			}
			if _, err := uuid.Parse(ops[1].ID); err != nil {
				t.Fatalf("ID пополнения: %v", err)
			}
			if !ops[0].CreatedAt.After(ops[1].CreatedAt) {
				t.Fatal("новые операции должны идти первыми")
			}
			rec = request(t, "GET", "/api/clients/anya/operations?limit=1", "", 200)
			if err := json.Unmarshal(rec.Body.Bytes(), &ops); err != nil || len(ops) != 1 || ops[0].Type != "withdrawal" {
				t.Fatalf("limit=1: %s, %v", rec.Body, err)
			}
			clk.Advance(time.Hour)
			if _, err := pool.Exec(ctx, `INSERT INTO deposits (id, client_id, amount, created_at) VALUES ('dep_legacy', 'anya', 10000, $1)`, clk.Now()); err != nil {
				t.Fatal(err)
			}
			rec = request(t, "GET", "/api/clients/anya/operations", "", 200)
			if err := json.Unmarshal(rec.Body.Bytes(), &ops); err != nil {
				t.Fatal(err)
			}
			if len(ops) != 3 || !strings.HasPrefix(ops[0].ID, "dep_") || ops[0].Type != "transfer_in" || ops[0].DecisionID != nil {
				t.Fatalf("ручное пополнение в истории: %s", rec.Body)
			}

		})
	}
	id := uuid.NewString()
	if err := store.InTx(ctx, func(r *storage.Repo) error { return r.EnsureClient(ctx, id) }); err != nil {
		t.Fatal(err)
	}
	defer store.InTx(ctx, func(r *storage.Repo) error { return r.DeleteClient(ctx, id) })
	if rec := request(t, "GET", "/api/clients/"+id+"/operations", "", 200); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("пустая история: %s", rec.Body)
	}
	request(t, "GET", "/api/clients/"+uuid.NewString()+"/operations", "", 404)
	for _, limit := range []string{"abc", "0", "-1", "101"} {
		request(t, "GET", "/api/clients/anya/operations?limit="+limit, "", 400)
	}
	// Создание вопроса само по себе не является движением денег.
	res, err := runner.Run(ctx, "income_drop")
	if err != nil || res.Decision == nil {
		t.Fatalf("сценарий: %v", err)
	}
	if rec := request(t, "GET", "/api/clients/anya/operations", "", 200); strings.TrimSpace(rec.Body.String()) != "[]" {
		t.Fatalf("неисполненное предложение попало в историю: %s", rec.Body)
	}
}
