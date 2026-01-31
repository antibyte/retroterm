package middleware

import (
	"net/http"
)

// SecurityHeaders middleware adds security-related HTTP headers to responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Enable XSS protection (for older browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		// Allows scripts/styles from self, inline, and specific external sources
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://fonts.googleapis.com https://webonastick.com; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com; " +
			"font-src 'self' https://fonts.gstatic.com https://webonastick.com; " +
			"img-src 'self' data:; " +
			"connect-src 'self' ws: wss:; " +
			"object-src 'none'; " +
			"frame-ancestors 'self';"
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}
