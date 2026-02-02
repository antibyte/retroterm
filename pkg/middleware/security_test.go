package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a simple handler that we'll wrap
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Wrap the handler with our middleware
	handler := SecurityHeaders(nextHandler)

	// Create a request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(w, req)

	// Check response body
	if w.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got '%s'", w.Body.String())
	}

	// Check headers
	expectedHeaders := map[string]string{
		"X-XSS-Protection":       "1; mode=block",
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "SAMEORIGIN",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expected := range expectedHeaders {
		got := w.Header().Get(header)
		if got != expected {
			t.Errorf("Expected header %s to be '%s', got '%s'", header, expected, got)
		}
	}

	// Check CSP specifically as it is long
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header is missing")
	}

	expectedCSPParts := []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
		"font-src 'self' https://fonts.gstatic.com https://webonastick.com",
		"connect-src 'self' ws: wss:",
		"img-src 'self' data:",
	}

	for _, part := range expectedCSPParts {
		if !strings.Contains(csp, part) {
			t.Errorf("Content-Security-Policy missing part: '%s'. Got: '%s'", part, csp)
		}
	}
}
