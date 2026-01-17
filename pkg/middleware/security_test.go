package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a dummy handler that the middleware will wrap
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the dummy handler with our middleware
	handler := SecurityHeaders(dummyHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(w, req)

	// Check if the headers are present
	headers := w.Header()

	// Check Content-Security-Policy
	csp := headers.Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header missing")
	}
	expectedCSPDirectives := []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com",
		"img-src 'self' data: blob:",
		"font-src 'self' https://fonts.gstatic.com https://webonastick.com",
		"worker-src 'self' blob:",
		"connect-src 'self' ws: wss:",
	}
	for _, directive := range expectedCSPDirectives {
		if !strings.Contains(csp, directive) {
			t.Errorf("Content-Security-Policy missing directive: %s", directive)
		}
	}

	// Check X-Frame-Options
	if val := headers.Get("X-Frame-Options"); val != "SAMEORIGIN" {
		t.Errorf("Expected X-Frame-Options to be SAMEORIGIN, got %s", val)
	}

	// Check X-Content-Type-Options
	if val := headers.Get("X-Content-Type-Options"); val != "nosniff" {
		t.Errorf("Expected X-Content-Type-Options to be nosniff, got %s", val)
	}

	// Check Referrer-Policy
	if val := headers.Get("Referrer-Policy"); val != "strict-origin-when-cross-origin" {
		t.Errorf("Expected Referrer-Policy to be strict-origin-when-cross-origin, got %s", val)
	}

	// Check X-XSS-Protection
	if val := headers.Get("X-XSS-Protection"); val != "1; mode=block" {
		t.Errorf("Expected X-XSS-Protection to be 1; mode=block, got %s", val)
	}
}
