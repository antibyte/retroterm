package middleware

import (
	"net/http"
)

// SecurityHeaders middleware adds security-related headers to the response
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Content Security Policy
		// Allows scripts/styles from self and specific external sources
		// unsafe-inline and unsafe-eval are currently needed for the application
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com; " +
			"font-src 'self' https://fonts.gstatic.com https://webonastick.com data:; " +
			"connect-src 'self' ws: wss:; " +
			"img-src 'self' data:; " +
			"media-src 'self'; " +
			"object-src 'none'; " +
			"frame-ancestors 'none';"
		w.Header().Set("Content-Security-Policy", csp)

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// HTTP Strict Transport Security (HSTS)
		// This is ignored by browsers on HTTP, but enforced on HTTPS
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")

		next.ServeHTTP(w, r)
	})
}
