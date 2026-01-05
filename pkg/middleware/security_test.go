package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a simple handler to wrap
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Wrap the handler with the security middleware
	secureHandler := WithSecurityHeaders(handler)

	// Create a test request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	w := httptest.NewRecorder()

	// Execute the request
	secureHandler.ServeHTTP(w, req)

	// Check response status
	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %v", w.Code)
	}

	// Helper function to check header existence and value
	checkHeader := func(key, expectedValue string) {
		val := w.Header().Get(key)
		if val == "" {
			t.Errorf("Header %s is missing", key)
		} else if expectedValue != "" && val != expectedValue {
			t.Errorf("Header %s: expected %s, got %s", key, expectedValue, val)
		}
	}

	// Verify headers
	checkHeader("X-Content-Type-Options", "nosniff")
	checkHeader("X-Frame-Options", "DENY")
	checkHeader("X-XSS-Protection", "1; mode=block")
	checkHeader("Referrer-Policy", "strict-origin-when-cross-origin")
	checkHeader("Content-Security-Policy", "") // Just check it exists

	// Verify CSP content partially (flexible matching)
	csp := w.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header is missing")
	}

	requiredDirectives := []string{
		"default-src 'self'",
		"object-src 'none'",
		"script-src 'self'",
	}

	for _, directive := range requiredDirectives {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP missing directive: %s", directive)
		}
	}

	// HSTS should NOT be set for non-TLS request
	if w.Header().Get("Strict-Transport-Security") != "" {
		t.Error("Strict-Transport-Security should not be set for HTTP request")
	}
}

func TestHSTSHeader(t *testing.T) {
	// Create a simple handler to wrap
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	// Wrap the handler
	secureHandler := WithSecurityHeaders(handler)

	// Create a test request simulates HTTPS
	req := httptest.NewRequest("GET", "https://example.com/foo", nil)
	// Manually set TLS state to simulate HTTPS connection
	req.TLS = &tls.ConnectionState{}

	w := httptest.NewRecorder()

	// Execute the request
	secureHandler.ServeHTTP(w, req)

	// Verify HSTS header
	hsts := w.Header().Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("Strict-Transport-Security header is missing for HTTPS request")
	}
	if !strings.Contains(hsts, "max-age=") {
		t.Errorf("Invalid HSTS header: %s", hsts)
	}
}
