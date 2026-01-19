package middleware

import (
	"net/http"
)

// SecurityHeadersMiddleware adds security-related headers to all responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Enable XSS protection (for older browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Control information leakage in Referer header
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		// Allowing unsafe-inline and unsafe-eval because the frontend architecture requires it
		// Allowing data: and blob: for images and file operations
		// Allowing ws: and wss: for WebSocket connections
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"connect-src 'self' ws: wss:;"

		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}
