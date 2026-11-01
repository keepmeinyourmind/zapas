package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter ограничивает количество запросов с одного IP
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter создает новый лимитер
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Middleware оборачивает http.Handler и применяет rate limiting
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Извлекаем только IP без порта
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// Если не удалось разделить, используем весь RemoteAddr
			ip = r.RemoteAddr
		}

		rl.mu.Lock()
		defer rl.mu.Unlock()

		// Очищаем старые запросы за пределами временного окна
		cutoff := time.Now().Add(-rl.window)
		var recent []time.Time
		for _, t := range rl.requests[ip] {
			if t.After(cutoff) {
				recent = append(recent, t)
			}
		}

		// Проверяем, не превышен ли лимит
		if len(recent) >= rl.limit {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too Many Requests. Please try again later.", http.StatusTooManyRequests)
			return
		}

		// Добавляем текущий запрос в список
		rl.requests[ip] = append(recent, time.Now())

		// Передаем управление следующему обработчику
		next.ServeHTTP(w, r)
	})
}

// Cleanup запускает фоновую очистку старых записей
func (rl *RateLimiter) Cleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window)
		for ip, times := range rl.requests {
			var recent []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					recent = append(recent, t)
				}
			}
			if len(recent) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = recent
			}
		}
		rl.mu.Unlock()
	}
}
