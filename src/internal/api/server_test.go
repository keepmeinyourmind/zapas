package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Эти проверки срабатывают до обращения к сервису, поэтому сервис и БД не нужны.
func testServer() http.Handler {
	s := NewServer(nil, nil, nil, "http://localhost:5173", slog.New(slog.NewTextHandler(io.Discard, nil)))
	return s.Routes()
}

func TestRequestGuards(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{
			name: "тело больше 1 МБ — 400", method: http.MethodPost, path: "/api/transactions",
			body: `{"id":"` + strings.Repeat("x", 2<<20) + `"}`, want: http.StatusBadRequest,
		},
		{
			name: "id решения не UUID — 404, а не 500", method: http.MethodPost, path: "/api/decisions/abc/answer",
			body: `{"choice":"skip"}`, want: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			testServer().ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body)))
			if rec.Code != tt.want {
				t.Errorf("код %d, want %d: %s", rec.Code, tt.want, rec.Body)
			}
		})
	}
}

// В CORS объявляем только методы, которые есть в API: DELETE-маршрутов нет.
func TestCORSAllowsOnlyUsedMethods(t *testing.T) {
	rec := httptest.NewRecorder()
	testServer().ServeHTTP(rec, httptest.NewRequest(http.MethodOptions, "/api/transactions", nil))
	if got := rec.Header().Get("Access-Control-Allow-Methods"); strings.Contains(got, "DELETE") {
		t.Errorf("Allow-Methods = %q — DELETE не используется", got)
	}
}
