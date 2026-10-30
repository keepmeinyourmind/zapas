package middleware

import "net/http"

// SecurityHeaders добавляет заголовки безопасности ко всем HTTP-ответам
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Защита от MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")
		
		// Защита от clickjacking (запрет встраивания в iframe)
		w.Header().Set("X-Frame-Options", "DENY")
		
		// Защита от XSS в старых браузерах
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		
		// Принудительный HTTPS (HSTS)
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		
		// Политика безопасности контента (CSP)
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		
		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		
		// Permissions Policy
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		
		next.ServeHTTP(w, r)
	})
}