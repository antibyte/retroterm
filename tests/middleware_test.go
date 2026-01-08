package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antibyte/retroterm/pkg/middleware"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create a simple handler that we can wrap
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Wrap the handler with the middleware
	handler := middleware.SecurityHeadersMiddleware(nextHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(w, req)

	// Check response code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	// Check headers
	headers := w.Header()

	expectedHeaders := map[string]string{
		"Content-Security-Policy":   "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self';",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}

	for key, expectedValue := range expectedHeaders {
		if value := headers.Get(key); value != expectedValue {
			t.Errorf("Expected header %s to be %q, got %q", key, expectedValue, value)
		}
	}
}
