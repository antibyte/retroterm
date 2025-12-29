package middleware

import (
	"net/http"
)

// WithSecurityHeaders wraps an http.Handler and adds security headers to the response.
// It implements defense-in-depth by adding multiple security layers.
func WithSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking by denying framing (or allowing same origin if needed)
		// Using DENY is safest; if framing is needed, change to SAMEORIGIN
		w.Header().Set("X-Frame-Options", "DENY")

		// Enable XSS protection in older browsers (blocks the response if attack detected)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Control referrer information sent to other sites
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy (CSP)
		// This is a defense-in-depth measure against XSS and data injection.
		// We use a permissive policy initially to avoid breaking existing functionality,
		// but it still provides significant protection over having no CSP.
		// 'unsafe-inline' and 'unsafe-eval' are kept because the application (TinyBASIC/RetroTerm)
		// likely relies on inline scripts/styles or dynamic code evaluation.
		// Images and media are allowed from self and data URIs.
		// Connect (WebSockets/XHR) allowed to self and standard WS schemes.
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data:; " +
			"font-src 'self'; " +
			"connect-src 'self' ws: wss:; " +
			"media-src 'self' blob:; " +
			"object-src 'none';"

		w.Header().Set("Content-Security-Policy", csp)

		// HSTS (HTTP Strict Transport Security)
		// Ensures that browsers always use HTTPS.
		// Only set if the connection is TLS/HTTPS to avoid locking out HTTP-only dev environments.
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}

		next.ServeHTTP(w, r)
	})
}
