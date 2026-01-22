package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	// Create a dummy handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with middleware
	secureHandler := SecurityHeaders(handler)

	// Create a request
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	secureHandler.ServeHTTP(rec, req)

	// Check headers
	headers := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"Referrer-Policy":           "strict-origin-when-cross-origin",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}

	for header, expectedValue := range headers {
		value := rec.Header().Get(header)
		if value == "" {
			t.Errorf("Header %s is missing", header)
			continue
		}
		if value != expectedValue {
			t.Errorf("Header %s: expected %q, got %q", header, expectedValue, value)
		}
	}

	// Check specific critical CSP directives
	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Error("Content-Security-Policy header is missing")
	} else {
		mustHave := []string{
			"default-src 'self'",
			"frame-ancestors 'none'",
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
			"object-src 'none'",
		}
		for _, part := range mustHave {
			if !strings.Contains(csp, part) {
				t.Errorf("CSP missing directive: %s. Full CSP: %s", part, csp)
			}
		}
	}
}
