package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeaders adds various security headers to the response
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		// Allow scripts from self and unsafe-inline/eval (needed for retroterm)
		// Allow styles from self and unsafe-inline
		// Allow fonts from self, Google Fonts, and webonastick.com
		// Allow connections to self and ws/wss
		csp := []string{
			"default-src 'self'",
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com",
			"font-src 'self' https://fonts.gstatic.com https://webonastick.com",
			"img-src 'self' data:",
			"connect-src 'self' ws: wss:",
			"media-src 'self'",
			"object-src 'none'",
			"frame-ancestors 'self'",
		}
		w.Header().Set("Content-Security-Policy", strings.Join(csp, "; "))

		next.ServeHTTP(w, r)
	})
}
