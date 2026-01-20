package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related headers to the response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy
		// Allow:
		// - 'self' for everything by default
		// - 'unsafe-inline' and 'unsafe-eval' for scripts (required by current frontend architecture)
		// - External fonts from Google and webonastick.com
		// - Data and Blob URIs for images and media
		// - WebSockets (ws: and wss:)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com; font-src 'self' https://fonts.gstatic.com https://webonastick.com; img-src 'self' data: blob:; connect-src 'self' ws: wss:; media-src 'self' data: blob:; object-src 'none'; frame-ancestors 'none';")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}
