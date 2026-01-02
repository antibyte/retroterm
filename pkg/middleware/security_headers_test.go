package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a mock handler that we can wrap
	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap the mock handler with our middleware
	handler := SecurityHeaders(mockHandler)

	// Create a request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(w, req)

	// Check response code
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check security headers
	headers := map[string]string{
		"Content-Security-Policy":  "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self' ws: wss:;",
		"X-Frame-Options":          "DENY",
		"X-Content-Type-Options":   "nosniff",
		"Referrer-Policy":          "strict-origin-when-cross-origin",
	}

	for key, expectedValue := range headers {
		if value := w.Header().Get(key); value != expectedValue {
			t.Errorf("Header %s: expected %q, got %q", key, expectedValue, value)
		}
	}
}
