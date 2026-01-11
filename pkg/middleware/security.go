package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeadersMiddleware adds security-related headers to all responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy (CSP)
		// We allow 'unsafe-inline' and 'unsafe-eval' because the retroterm app likely relies on them
		// for its terminal emulation and game logic. Ideally these should be removed in the future.
		// We also allow connections to self and ws/wss for websockets.
		csp := "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss:;"
		w.Header().Set("Content-Security-Policy", csp)

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// HTTP Strict Transport Security (HSTS) - only if HTTPS
		if r.TLS != nil || strings.ToLower(r.Header.Get("X-Forwarded-Proto")) == "https" {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
