package middleware

import (
	"net/http"
	"os"
	"strings"
)

// ProductionGuard блокирует доступ к отладочным эндпоинтам в production-окружении
func ProductionGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Если окружение production, блокируем debug и scenario роуты
		if os.Getenv("ENV") == "production" {
			if strings.HasPrefix(r.URL.Path, "/api/debug/") || strings.HasPrefix(r.URL.Path, "/api/scenarios/") {
				http.Error(w, "Forbidden: debug endpoints are disabled in production", http.StatusForbidden)
				return
			}
		}

		// В остальных случаях (local, dev) пропускаем запрос дальше
		next.ServeHTTP(w, r)
	})
}
