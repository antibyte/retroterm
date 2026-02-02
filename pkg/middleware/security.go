package middleware

import (
	"net/http"
)

// SecurityHeaders adds security-related headers to the response
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// X-XSS-Protection: 1; mode=block
		// Enables XSS filtering. Rather than sanitizing the page, the browser will prevent rendering of the page if an attack is detected.
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// X-Content-Type-Options: nosniff
		// Prevents the browser from interpreting files as a different MIME type to what is specified by the Content-Type HTTP header
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// X-Frame-Options: SAMEORIGIN
		// The page can only be displayed in a frame on the same origin as the page itself.
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")

		// Referrer-Policy: strict-origin-when-cross-origin
		// Send a full URL when performing a same-origin request, but only send the origin of the document for other cases.
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content-Security-Policy
		// A computer security standard introduced to prevent cross-site scripting (XSS), clickjacking and other code injection attacks
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; " +
			"font-src 'self' https://fonts.gstatic.com https://webonastick.com; " +
			"connect-src 'self' ws: wss:; " +
			"img-src 'self' data:;"
		w.Header().Set("Content-Security-Policy", csp)

		next.ServeHTTP(w, r)
	})
}
