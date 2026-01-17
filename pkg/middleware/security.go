package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related headers to the response
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy (CSP)
		// Allows scripts from self, inline, and eval (needed for legacy/libs)
		// Allows styles from self, inline, Google Fonts, and Webonastick
		// Allows images from self, data URIs, and blob URIs
		// Allows fonts from self, Google Fonts (gstatic), and Webonastick
		// Allows WebSocket connections to self (ws: and wss:)
		// Allows workers from self and blob (common for physics engines etc)
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self' https://fonts.gstatic.com https://webonastick.com; " +
			"worker-src 'self' blob:; " +
			"connect-src 'self' ws: wss:;"

		w.Header().Set("Content-Security-Policy", csp)

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Enable XSS protection (for older browsers)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		next.ServeHTTP(w, r)
	})
}
