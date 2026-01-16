package middleware

import (
	"net/http"
)

// WithSecurityHeaders wraps an http.Handler to add security headers to the response.
// This implements defense-in-depth measures against common web vulnerabilities.
func WithSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Control information leakage in Referer header
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy
		// Restrictive default, but allow necessary retro-terminal features:
		// - unsafe-inline/eval: Required for retro graphics/audio libraries and dynamic JS loading
		// - blob:/data:: Required for audio generation (SID player) and images
		// - ws:/wss:: Required for terminal WebSocket connection
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' blob:; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' ws: wss:; " +
			"object-src 'none'; " +
			"base-uri 'self';"
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}
