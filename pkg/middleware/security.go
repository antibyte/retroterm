package middleware

import (
	"net/http"
)

// SecurityHeaders adds HTTP security headers to the response
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Enable XSS protection in legacy browsers
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		// Allowing 'unsafe-inline' and 'unsafe-eval' is necessary for the current architecture
		// (frontend likely uses eval() or inline scripts)
		// Added connect-src for WebSockets (ws: wss:)
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self'; " +
			"connect-src 'self' ws: wss:; " +
			"object-src 'none'; " +
			"media-src 'self'; " +
			"frame-ancestors 'none';"

		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}
