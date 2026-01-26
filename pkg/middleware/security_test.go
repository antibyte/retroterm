package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a dummy handler
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap it with the middleware
	securedHandler := SecurityHeaders(dummyHandler)

	// Create a request
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	securedHandler.ServeHTTP(rr, req)

	// Check the headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "SAMEORIGIN",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Content-Security-Policy": "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com; font-src 'self' https://fonts.gstatic.com https://webonastick.com; img-src 'self' data:; connect-src 'self' ws: wss:; media-src 'self'; object-src 'none'; frame-ancestors 'self'",
	}

	for header, expectedValue := range expectedHeaders {
		if got := rr.Header().Get(header); got != expectedValue {
			t.Errorf("Header %s: expected %q, got %q", header, expectedValue, got)
		}
	}
}
