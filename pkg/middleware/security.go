package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders adds various security-related HTTP headers to the response
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy
		// Allow scripts/styles from 'self', 'unsafe-inline', 'unsafe-eval' (as per requirements)
		// Allow fonts from Google and webonastick.com
		// Allow WebSockets
		// Allow images from 'self' and data:
		csp := []string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
			"font-src 'self' https://fonts.gstatic.com https://webonastick.com data:",
			"img-src 'self' data:",
			"connect-src 'self' ws: wss:",
			"frame-ancestors 'none'", // Prevent clickjacking
			"object-src 'none'",      // Prevent Flash etc.
		}
		w.Header().Set("Content-Security-Policy", strings.Join(csp, "; "))

		// Prevent sniffing of content type
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking (legacy browsers)
		w.Header().Set("X-Frame-Options", "DENY")

		// Referrer Policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// HSTS (Strict Transport Security) - 1 year
		// Only effective if served over HTTPS, but good to have
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		next.ServeHTTP(w, r)
	})
}
