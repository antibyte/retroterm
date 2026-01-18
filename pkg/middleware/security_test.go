package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a dummy handler
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap it with the middleware
	handler := SecurityHeaders(dummyHandler)

	// Create a request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(w, req)

	// Check headers
	headers := w.Header()

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "SAMEORIGIN",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for key, expectedValue := range expectedHeaders {
		if value := headers.Get(key); value != expectedValue {
			t.Errorf("Header %s: expected %q, got %q", key, expectedValue, value)
		}
	}

	// Check CSP specifically as it might vary slightly in formatting
	csp := headers.Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header is missing")
	}

	// Verify essential CSP directives
	requiredDirectives := []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
		"object-src 'none'",
		"frame-ancestors 'none'",
	}

	for _, directive := range requiredDirectives {
		if !strings.Contains(csp, directive) {
			t.Errorf("Content-Security-Policy missing directive: %s", directive)
		}
	}
}
