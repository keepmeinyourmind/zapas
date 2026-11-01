package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/keepmeinyourmind/zapas/src/internal/clock"
	"github.com/keepmeinyourmind/zapas/src/internal/config"
	"github.com/keepmeinyourmind/zapas/src/internal/domain"
	"github.com/keepmeinyourmind/zapas/src/internal/service"
	"github.com/keepmeinyourmind/zapas/src/internal/storage"
)

func TestGetRuleReturnsAllSettings(t *testing.T) {
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
	store := storage.New(pool)
	id := uuid.NewString()
	want := domain.SavingsRule{ClientID: id, GoalName: "Подушка", GoalAmount: 9000000, TermMonths: 24, SavePercent: 17, MinIncome: 45000, MinTransfer: 15000, DeclaredIncome: 4200000, Paused: true, QuietFrom: "22:30", QuietTo: "08:15", Timezone: "Asia/Vladivostok"}
	err = store.InTx(ctx, func(r *storage.Repo) error {
		if err := r.EnsureClient(ctx, id); err != nil {
			return err
		}
		return r.UpsertRule(ctx, want)
	})
	if err != nil {
		t.Fatal(err)
	}
	defer store.InTx(ctx, func(r *storage.Repo) error { return r.DeleteClient(ctx, id) })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	clk := clock.New()
	svc := service.New(store, config.Load(), clk, log)
	handler := NewServer(svc, nil, clk, "http://localhost:5173", log).Routes()
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest("GET", "/api/clients/"+id+"/rule", nil))
	if rec.Code != 200 {
		t.Fatalf("HTTP %d: %s", rec.Code, rec.Body)
	}
	var got domain.SavingsRule
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("получено %+v, ожидали %+v", got, want)
	}
	for _, missing := range []string{uuid.NewString(), id + "-without-rule"} {
		if err := store.InTx(ctx, func(r *storage.Repo) error { return r.EnsureClient(ctx, id+"-without-rule") }); err != nil {
			t.Fatal(err)
		}
		rec = httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest("GET", "/api/clients/"+missing+"/rule", nil))
		if rec.Code != 404 {
			t.Fatalf("отсутствующее правило: HTTP %d", rec.Code)
		}
	}
	if err := store.InTx(ctx, func(r *storage.Repo) error { return r.DeleteClient(ctx, id+"-without-rule") }); err != nil {
		t.Fatal(err)
	}
}
