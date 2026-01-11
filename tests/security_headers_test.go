package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/antibyte/retroterm/pkg/middleware"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Wrap it with the security middleware
	secureHandler := middleware.SecurityHeadersMiddleware(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	// Serve the request
	secureHandler.ServeHTTP(w, req)

	// Check headers
	headers := w.Header()

	checks := []struct {
		header string
		value  string
	}{
		{"X-Frame-Options", "DENY"},
		{"X-Content-Type-Options", "nosniff"},
		{"Referrer-Policy", "strict-origin-when-cross-origin"},
		{"Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws: wss:;"},
	}

	for _, check := range checks {
		if val := headers.Get(check.header); val != check.value {
			t.Errorf("Expected %s to be %q, but got %q", check.header, check.value, val)
		}
	}
}

func TestHSTSHeader(t *testing.T) {
	// Create a simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Wrap it with the security middleware
	secureHandler := middleware.SecurityHeadersMiddleware(handler)

	// Test 1: Plain HTTP (should NOT have HSTS)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	secureHandler.ServeHTTP(w, req)
	if headers := w.Header(); headers.Get("Strict-Transport-Security") != "" {
		t.Error("HSTS header should not be present on non-HTTPS requests")
	}

	// Test 2: HTTPS (should HAVE HSTS)
	reqTLS := httptest.NewRequest("GET", "/", nil)
	// Simulate TLS via X-Forwarded-Proto since mocking r.TLS is harder
	reqTLS.Header.Set("X-Forwarded-Proto", "https")

	wTLS := httptest.NewRecorder()
	secureHandler.ServeHTTP(wTLS, reqTLS)
	if val := wTLS.Header().Get("Strict-Transport-Security"); val != "max-age=31536000; includeSubDomains" {
		t.Errorf("Expected HSTS header to be set, got %q", val)
	}
}
