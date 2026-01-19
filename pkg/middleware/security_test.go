package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create a dummy handler that we'll wrap
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap it with our middleware
	middleware := SecurityHeadersMiddleware(handler)

	// Create a request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Serve the request
	middleware.ServeHTTP(w, req)

	// Check headers
	headers := w.Result().Header

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "SAMEORIGIN",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for key, expected := range expectedHeaders {
		if got := headers.Get(key); got != expected {
			t.Errorf("Expected %s: %s, got %s", key, expected, got)
		}
	}

	// Check CSP separately
	csp := headers.Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header is missing")
	}

	requiredDirectives := []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob:",
		"connect-src 'self' ws: wss:",
	}

	for _, directive := range requiredDirectives {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP missing directive part: %s. Got: %s", directive, csp)
		}
	}
}
