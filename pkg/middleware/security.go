package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related HTTP headers to the response
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Protects against MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Protects against clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Protects against XSS (legacy browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Controls information in Referer header
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		// Allows scripts/styles from self and specific external sources
		// unsafe-inline/eval required for current frontend architecture
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com; font-src 'self' https://fonts.gstatic.com https://webonastick.com; img-src 'self' data:; connect-src 'self' ws: wss: https://webonastick.com; object-src 'none'; base-uri 'self';")

		next.ServeHTTP(w, r)
	})
}
