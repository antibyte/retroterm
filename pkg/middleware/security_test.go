package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a dummy handler that we wrap with our middleware
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap the dummy handler with SecurityHeaders middleware
	handler := SecurityHeaders(dummyHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	handler.ServeHTTP(rr, req)

	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check headers
	expectedHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Strict-Transport-Security": "max-age=63072000; includeSubDomains",
	}

	for header, expectedValue := range expectedHeaders {
		if value := rr.Header().Get(header); value != expectedValue {
			t.Errorf("handler returned wrong %s header: got %v want %v",
				header, value, expectedValue)
		}
	}

	// Check CSP header separately as it is long and might have formatting differences
	csp := rr.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Errorf("handler returned empty Content-Security-Policy header")
	}

	expectedCSPDirectives := []string{
		"default-src 'self'",
		"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
		"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com https://webonastick.com",
		"font-src 'self' https://fonts.gstatic.com https://webonastick.com data:",
		"connect-src 'self' ws: wss:",
		"img-src 'self' data:",
		"media-src 'self'",
		"object-src 'none'",
		"frame-ancestors 'none'",
	}

	for _, directive := range expectedCSPDirectives {
		if !strings.Contains(csp, directive) {
			t.Errorf("CSP header missing directive %q. Got: %s", directive, csp)
		}
	}
}
