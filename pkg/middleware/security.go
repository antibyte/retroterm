package middleware

import (
	"net/http"
)

// SecurityHeadersMiddleware adds security headers to the response
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy
		// Allow scripts from 'self', inline scripts (needed for retroterm), and eval (needed for emulator)
		// Restrict object-src to 'none' to prevent Flash/Java applets
		// Set base-uri to 'self' to prevent base tag hijacking
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' ws: wss:; " +
			"object-src 'none'; " +
			"base-uri 'self';"
		w.Header().Set("Content-Security-Policy", csp)

		// HTTP Strict Transport Security (HSTS)
		// Enforce HTTPS for 1 year, include subdomains
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// X-Content-Type-Options
		// Prevent MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// X-Frame-Options
		// Prevent clickjacking by disallowing framing
		w.Header().Set("X-Frame-Options", "DENY")

		// Referrer-Policy
		// Limit referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		next.ServeHTTP(w, r)
	})
}
