package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related headers to all responses
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy
		// Allows scripts and styles from 'self' and uses 'unsafe-inline' and 'unsafe-eval'
		// which are currently required by the application architecture.
		// Restricts other resources to 'self' or data URIs for images.
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self' ws: wss:;")

		// Prevent clickjacking by not allowing the site to be embedded in iframes
		// unless from the same origin.
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Control referrer information sent to other sites
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Enable HSTS if using HTTPS (will be ignored over HTTP, but good practice)
		// Max-age is set to 1 year (31536000 seconds)
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
