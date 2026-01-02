package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy
		// Allows 'unsafe-inline' and 'unsafe-eval' for scripts/styles to maintain compatibility with the frontend
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self' ws: wss:;")

		// Prevent page from being displayed in an iframe
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Control how much referrer information is sent
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// If next is nil, it means we should use the DefaultServeMux
		if next == nil {
			http.DefaultServeMux.ServeHTTP(w, r)
		} else {
			next.ServeHTTP(w, r)
		}
	})
}
